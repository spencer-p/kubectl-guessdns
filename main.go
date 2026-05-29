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
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: kubectl-guessdns [query-terms]")
		os.Exit(1)
	}
	query := strings.ToLower(strings.Join(args, "-"))

	var kubeconfigPath string
	if *kubeconfig != "" {
		kubeconfigPath = *kubeconfig
	} else if env := os.Getenv("KUBECONFIG"); env != "" {
		kubeconfigPath = env
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	bestMatch := ""
	bestScore := -1.0

	// Check Services
	svcs, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing services: %v", err)
	}

	for _, svc := range svcs.Items {
		score := getSimilarity(query, strings.ToLower(svc.Name))
		if score > bestScore && score >= threshold {
			bestScore = score
			bestMatch = fmt.Sprintf("%s.%s.svc.%s", svc.Name, svc.Namespace, clusterDomain)
		}
	}

	// Check Pods
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing pods: %v", err)
	}

	for _, pod := range pods.Items {
		score := getSimilarity(query, strings.ToLower(pod.Name))
		if score > bestScore && score >= podThreshold {
			if pod.Status.PodIP != "" {
				bestScore = score
				ipDashed := strings.ReplaceAll(pod.Status.PodIP, ".", "-")
				bestMatch = fmt.Sprintf("%s.%s.pod.%s", ipDashed, pod.Namespace, clusterDomain)
			}
		}
	}

	if bestMatch != "" {
		fmt.Println(bestMatch)
		return
	}

	fmt.Fprintln(os.Stderr, "No likely candidate found.")
	os.Exit(1)
}
