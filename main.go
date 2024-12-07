package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yankeguo/rg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type NamespacedName struct {
	Namespace string
	Name      string
}

const (
	ContainerIDPrefixContainerd = "containerd://"

	KeyEphemeralStorage = "com.yankeguo.knowstore/ephemeral-storage"
)

var (
	optNodeName           = strings.TrimSpace(os.Getenv("NODE_NAME"))
	optKubeconfig         string
	optContainerdStateDir string
	optInterval           time.Duration
)

func defaultKubeconfig() string {
	if dirHome, err := os.UserHomeDir(); err == nil && dirHome != "" {
		kubeconfig := filepath.Join(dirHome, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err == nil {
			return kubeconfig
		}
	}
	return ""
}

func main() {
	flag.StringVar(&optKubeconfig, "kubeconfig", defaultKubeconfig(), "the kubeconfig file, if empty, use in-cluster config")
	flag.StringVar(&optContainerdStateDir, "containerd.state.dir", "/run/containerd", "the containerd state dir")
	flag.DurationVar(&optInterval, "interval", 10*time.Minute, "the interval to refresh, set to 0 to run once")
	flag.Parse()

loop:
	if err := once(); err != nil {
		log.Println("execution failed:", err.Error())
	}

	if optInterval > 0 {
		time.Sleep(optInterval)
		goto loop
	}
}

func once() (err error) {
	defer rg.Guard(&err)

	ctx := context.Background()

	client := rg.Must(createKubernetesClient())

	dir := filepath.Join(optContainerdStateDir, "io.containerd.runtime.v2.task", "k8s.io")

	entries := rg.Must(os.ReadDir(dir))

	log.Println("Found", len(entries), "entries in", dir)

	containerIDs := rg.Must(listContainerIDs(ctx, client))

	log.Println("Found", len(containerIDs), "container IDs")

	results := map[NamespacedName]int64{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		item, ok := containerIDs[entry.Name()]
		if !ok {
			continue
		}

		log.Println("Calculating", item.Namespace+"/"+item.Name, ":", entry.Name())

		var result int64
		calculateDiskUsage(&result, filepath.Join(dir, entry.Name()))

		if result == 0 {
			continue
		}

		results[item] += result
	}

	now := time.Now().Format(time.RFC3339)

	for item, size := range results {
		buf := rg.Must(json.Marshal(map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]string{
					KeyEphemeralStorage: humanReadableSize(size) + ":" + now,
				},
			},
		}))

		if _, err := client.CoreV1().Pods(item.Namespace).Patch(ctx, item.Name, types.MergePatchType, buf, metav1.PatchOptions{}); err != nil {
			log.Println("Patch failed for", item.Namespace+"/"+item.Name, ":", err.Error())
		}
	}

	return
}

func listContainerIDs(ctx context.Context, client *kubernetes.Clientset) (containerIDs map[string]NamespacedName, err error) {
	defer rg.Guard(&err)

	containerIDs = make(map[string]NamespacedName)

	opts := metav1.ListOptions{}

	if optNodeName != "" {
		opts.FieldSelector = "spec.nodeName=" + optNodeName
	}

	for _, item := range rg.Must(client.CoreV1().Pods("").List(ctx, opts)).Items {
		for _, container := range item.Status.ContainerStatuses {
			if strings.HasPrefix(container.ContainerID, ContainerIDPrefixContainerd) {
				containerID := strings.TrimPrefix(container.ContainerID, ContainerIDPrefixContainerd)
				containerIDs[containerID] = NamespacedName{Namespace: item.Namespace, Name: item.Name}
			}
		}
		for _, container := range item.Status.InitContainerStatuses {
			if strings.HasPrefix(container.ContainerID, ContainerIDPrefixContainerd) {
				containerID := strings.TrimPrefix(container.ContainerID, ContainerIDPrefixContainerd)
				containerIDs[containerID] = NamespacedName{Namespace: item.Namespace, Name: item.Name}
			}
		}
	}

	return
}

func createKubernetesClient() (client *kubernetes.Clientset, err error) {
	defer rg.Guard(&err)

	var config *rest.Config
	if optKubeconfig == "" {
		config = rg.Must(rest.InClusterConfig())
	} else {
		config = rg.Must(clientcmd.BuildConfigFromFlags("", optKubeconfig))
	}
	client = rg.Must(kubernetes.NewForConfig(config))

	return
}

func calculateDiskUsage(out *int64, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			calculateDiskUsage(out, filepath.Join(dir, entry.Name()))
		} else {
			info, err := entry.Info()
			if err != nil {
				return
			}
			*out += info.Size()
		}
	}
	return
}

func humanReadableSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2fT", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2fG", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2fM", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2fK", float64(size)/KB)
	default:
		return fmt.Sprintf("%d", size)
	}
}
