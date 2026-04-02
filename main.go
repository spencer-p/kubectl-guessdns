package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/agnivade/levenshtein"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	// Similarity threshold (0.0 to 1.0). Higher is stricter.
	threshold     = 0.4
	podThreshold  = 0.1
	clusterDomain = "cluster.local"
)

func getSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}
	dist := levenshtein.ComputeDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: kubectl-guessdns [query-terms]")
		os.Exit(1)
	}
	query := strings.ToLower(strings.Join(args, "-"))

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	// Try Services first
	svcs, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing services: %v", err)
	}

	bestSvcMatch := ""
	bestSvcScore := -1.0

	for _, svc := range svcs.Items {
		score := getSimilarity(query, strings.ToLower(svc.Name))
		if score > bestSvcScore && score >= threshold {
			bestSvcScore = score
			bestSvcMatch = fmt.Sprintf("%s.%s.svc.%s", svc.Name, svc.Namespace, clusterDomain)
		}
	}

	if bestSvcMatch != "" {
		fmt.Println(bestSvcMatch)
		return
	}

	// If no service found, try Pods
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing pods: %v", err)
	}

	bestPodMatch := ""
	bestPodScore := -1.0

	for _, pod := range pods.Items {
		score := getSimilarity(query, strings.ToLower(pod.Name))
		if score > bestPodScore && score >= podThreshold {
			bestPodScore = score
			if pod.Status.PodIP != "" {
				ipDashed := strings.ReplaceAll(pod.Status.PodIP, ".", "-")
				bestPodMatch = fmt.Sprintf("%s.%s.pod.%s", ipDashed, pod.Namespace, clusterDomain)
			}
		}
	}

	if bestPodMatch != "" {
		fmt.Println(bestPodMatch)
		return
	}

	fmt.Fprintln(os.Stderr, "No likely candidate found.")
	os.Exit(1)
}
