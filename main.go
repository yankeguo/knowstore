package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/yankeguo/rg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ContainerIDPrefixContainerd = "containerd://"

	KeyEphemeralStorageUsage       = "com.yankeguo.knowstore/ephemeral-storage.usage"
	KeyEphemeralStorageUsagePretty = "com.yankeguo.knowstore/ephemeral-storage.usage-pretty"
	KeyEphemeralStorageUpdatedAt   = "com.yankeguo.knowstore/ephemeral-storage.updated-at"
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
	flag.DurationVar(&optInterval, "interval", 30*time.Minute, "the interval to refresh, set to 0 to run once")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		chSig := make(chan os.Signal, 1)
		signal.Notify(chSig, syscall.SIGILL, syscall.SIGTERM)

		sig := <-chSig
		log.Println("signal received:", sig.String())

		cancel()
	}()

start:
	if err := do(ctx); err != nil {
		log.Println("execution failed:", err.Error())
	} else {
		log.Println("execution succeeded")
	}

	if optInterval > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(optInterval):
			goto start
		}
	}
}

func do(ctx context.Context) (err error) {
	defer rg.Guard(&err)

	client := rg.Must(createKubernetesClient())

	dir := filepath.Join(optContainerdStateDir, "io.containerd.runtime.v2.task", "k8s.io")

	entries := rg.Must(os.ReadDir(dir))

	log.Println("Found", len(entries), "entries in", dir)

	resultSet := NewResultSet()

	rg.Must0(setupResultSet(ctx, resultSet, client))

	log.Println("Found", resultSet.Len(), "Pods")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		cid := entry.Name()
		if !resultSet.HasCID(cid) {
			continue
		}

		log.Println("Calculating", cid)

		var size int64
		calculateDiskUsage(&size, filepath.Join(dir, entry.Name()))

		item, ok := resultSet.SaveUsage(cid, size)
		if !ok {
			continue
		}

		total, complete := resultSet.GetUsage(item)
		if !complete {
			continue
		}

		if err := saveUsage(ctx, client, item, total); err != nil {
			log.Println("Failed saving usage for", item.Namespace+"/"+item.Name, ":", err.Error())
		} else {
			log.Println("Saved usage for", item.Namespace+"/"+item.Name, ":", prettyDiskUsage(total))
		}
	}

	// init containers may have already purged, so we have to save incomplete ones
	for _, item := range resultSet.List() {
		total, complete := resultSet.GetUsage(item)
		if complete {
			// ignore complete since we have saved it
			continue
		}

		if err := saveUsage(ctx, client, item, total); err != nil {
			log.Println("Failed saving usage for", item.Namespace+"/"+item.Name, ":", err.Error())
		} else {
			log.Println("Saved usage for", item.Namespace+"/"+item.Name, ":", prettyDiskUsage(total))
		}
	}

	return
}

func saveUsage(ctx context.Context, client *kubernetes.Clientset, item NamespacedName, usage int64) (err error) {
	defer rg.Guard(&err)

	buf := rg.Must(json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				KeyEphemeralStorageUsage:       strconv.FormatInt(usage, 10),
				KeyEphemeralStorageUsagePretty: prettyDiskUsage(usage),
				KeyEphemeralStorageUpdatedAt:   time.Now().Format(time.RFC3339),
			},
		},
	}))

	rg.Must(client.CoreV1().Pods(item.Namespace).Patch(ctx, item.Name, types.MergePatchType, buf, metav1.PatchOptions{}))

	return
}

func setupResultSet(ctx context.Context, recordSet *ResultSet, client *kubernetes.Clientset) (err error) {
	defer rg.Guard(&err)

	opts := metav1.ListOptions{}
	if optNodeName != "" {
		opts.FieldSelector = "spec.nodeName=" + optNodeName
	}

	for _, item := range rg.Must(client.CoreV1().Pods("").List(ctx, opts)).Items {
		for _, container := range item.Status.ContainerStatuses {
			if strings.HasPrefix(container.ContainerID, ContainerIDPrefixContainerd) {
				recordSet.AddCID(
					strings.TrimPrefix(container.ContainerID, ContainerIDPrefixContainerd),
					NamespacedName{Namespace: item.Namespace, Name: item.Name},
				)
			}
		}
		for _, container := range item.Status.InitContainerStatuses {
			if strings.HasPrefix(container.ContainerID, ContainerIDPrefixContainerd) {
				recordSet.AddCID(
					strings.TrimPrefix(container.ContainerID, ContainerIDPrefixContainerd),
					NamespacedName{Namespace: item.Namespace, Name: item.Name},
				)
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
