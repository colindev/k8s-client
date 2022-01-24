package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/jsonpath"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var ObjEncoder = jsonpath.New("output")

func main() {

	var (
		kubeconfig        string
		kubeconfigDefault = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		//clusterContext    string
		namespace      string
		resourceName   string
		formatJsonPath string
	)

	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfigDefault, "kube config path")
	flag.StringVar(&resourceName, "resource", "", "kube resource name")
	flag.StringVar(&namespace, "n", v1.NamespaceAll, "kube namespace")
	flag.StringVar(&formatJsonPath, "jsonpath", "{@}", "kube resource jsonpath")
	//flag.StringVar(&clusterContext, "context", "", "kube context (only use outside cluster)")
	flag.Parse()

	defer func() {
		if err := recover(); err != nil {
			log.Println(err, "\n", string(debug.Stack()))
		}
	}()

	err := ObjEncoder.Parse(formatJsonPath)
	if err != nil {
		panic(err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil && kubeconfig == kubeconfigDefault {
		log.Println(err)
		// try InClusterConfig
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		log.Println("use rest.InClusterConfig..")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	watchlist := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		resourceName,
		namespace,
		fields.Everything())

	var allResource = map[string]runtime.Object{
		"services": &v1.Service{},
		"pods":     &v1.Pod{},
		"nodes":    &v1.Node{},
	}

	t, exists := allResource[resourceName]
	if !exists {
		panic(fmt.Sprintf("resource %s not support\n", resourceName))
	}

	_, controller := cache.NewInformer(
		watchlist,
		t,
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				buf := bytes.NewBuffer(nil)
				fmt.Fprint(buf, "add ")
				encode(buf, obj)
				buf.Write([]byte{'\n'})
				notify(buf)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				buf := bytes.NewBuffer(nil)
				fmt.Fprint(buf, "update ")
				encode(buf, oldObj)
				fmt.Fprint(buf, " to ")
				encode(buf, newObj)
				buf.Write([]byte{'\n'})
				notify(buf)
			},
			DeleteFunc: func(obj interface{}) {
				buf := bytes.NewBuffer(nil)
				fmt.Fprint(buf, "delete ")
				encode(buf, obj)
				buf.Write([]byte{'\n'})
				notify(buf)
			},
		})

	stop := make(chan struct{}, 1)
	go controller.Run(stop)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM)
	<-sig
	stop <- struct{}{}
	time.Sleep(time.Second * 1)
}

func encode(w io.Writer, obj interface{}) error {
	return ObjEncoder.Execute(w, obj)
}

func notify(buf *bytes.Buffer) {
	io.Copy(os.Stdout, buf)
}
