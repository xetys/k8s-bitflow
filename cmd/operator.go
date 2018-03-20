// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/api/core/v1"
	"time"
)

// operatorCmd represents the operator command
var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "starts the bitflow collector operator",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("starting bitflow collector operator")
		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		clientset, err := K8SClient()
		if err != nil {
			panic(err.Error())
		}

		svc := NewBitflowService(clientset)
		updater := NewUpdater(svc)
		watcher := NewBitflowWatcher(svc)

		// This goroutine executes a blocking receive for
		// signals. When it gets one it'll print it out
		// and then notify the program that it can finish.
		go func() {
			sig := <-sigs
			fmt.Println()
			fmt.Println(sig)
			done <- true
		}()

		go func() {

			err := watcher.WaitForReadyPods()
			if err != nil {
				log.Fatalln(err)
			}

			// create the pod watcher
			podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", "", fields.Everything())
			if err != nil {
				log.Println(err)
			}

			// create the workqueue
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

			// Bind the workqueue to a cache with the help of an informer. This way we make sure that
			// whenever the cache is updated, the pod key is added to the workqueue.
			// Note that when we finally process the item from the workqueue, we might see a newer version
			// of the Pod than the version which was responsible for triggering the update.
			indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						queue.Add(key)
					}
				},
				UpdateFunc: func(old interface{}, new interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						queue.Add(key)
					}
				},
				DeleteFunc: func(obj interface{}) {
					// IndexerInformer uses a delta queue, therefore for deletes we have to use this
					// key function.
					key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
					if err == nil {
						queue.Add(key)
					}
				},
			}, cache.Indexers{})

			controller := NewController(queue, indexer, informer, updater)
			// Now let's start the controller
			stop := make(chan struct{})
			defer close(stop)

			// watch and control pods
			go controller.Run(1, stop)

			// check the state of the bitflow pods
			go watcher.WatchSync(30 * time.Second)

			// Wait forever
			select {}
			//for {
			//podChan, err := clientset.CoreV1().Pods("").Watch(v1.ListOptions{})
			//event := <- podChan.ResultChan()
			//pod := event.Object.(*v12.Pod)
			//log.Println(event.Type, pod.Name)
			//}
		}()
		<-done
		watcher.DeletePods()
		log.Println("exiting")
	},
}

func init() {
	rootCmd.AddCommand(operatorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// operatorCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// operatorCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
