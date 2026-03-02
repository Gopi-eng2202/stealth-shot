package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/time/rate"
)

func main() {
	inputFile := flag.String("f", "", "File containing subdomains (required)")
	outputDir := flag.String("o", "results", "Output directory for screenshots")
	threads := flag.Int("t", 3, "Number of concurrent browser instances")
	delay := flag.Int("d", 5, "Delay in seconds between requests per thread")
	proxy := flag.String("proxy", "", "Optional: Proxy server (e.g., socks5://127.0.0.1:9050)")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("\nUsage: go run main.go -f hosts.txt -o my_scan -t 3 -d 5")
		os.Exit(1)
	}

	os.MkdirAll(*outputDir, 0755)

	// 1. Setup Rate Limiter to stagger requests
	limiter := rate.NewLimiter(rate.Every(time.Duration(*delay)*time.Second), 1)

	// 2. Job Queue and Synchronization
	jobs := make(chan string)
	var wg sync.WaitGroup

	fmt.Printf("[!] Starting Stealth-Shot | Threads: %d | Delay: %ds | Proxy: %s\n\n", *threads, *delay, *proxy)

	// 3. Launch Workers
	for w := 1; w <= *threads; w++ {
		wg.Add(1)
		go worker(w, jobs, &wg, *outputDir, limiter, *proxy)
	}

	// 4. Feed Subdomains into the Queue
	file, err := os.Open(*inputFile)
	if err != nil {
		log.Fatalf("[-] Error opening input file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" {
			jobs <- domain
		}
	}

	close(jobs)
	wg.Wait()
	fmt.Println("\n[✔] Recon Complete. Screenshots saved in:", *outputDir)
}

func worker(id int, jobs <-chan string, wg *sync.WaitGroup, dir string, limiter *rate.Limiter, proxyAddr string) {
	defer wg.Done()

	// Base Evasion Flags
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"), // Removes 'navigator.webdriver'
		chromedp.Flag("incognito", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	)

	// Add Proxy if provided
	if proxyAddr != "" {
		opts = append(opts, chromedp.ProxyServer(proxyAddr))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	for domain := range jobs {
		limiter.Wait(context.Background()) // Respect the rate limit

		targetURL := domain
		if !strings.HasPrefix(targetURL, "http") {
			targetURL = "https://" + targetURL
		}

		fmt.Printf("[Worker %d] Shooting: %s\n", id, targetURL)

		// New context for every screenshot
		ctx, _ := chromedp.NewContext(allocCtx)
		ctx, timeoutCancel := context.WithTimeout(ctx, 40*time.Second)

		var buf []byte
		err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				// Final layer of JS stealth
				return chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil).Do(ctx)
			}),
			chromedp.Navigate(targetURL),
			chromedp.Sleep(10*time.Second), // Wait for Akamai challenges to solve
			chromedp.FullScreenshot(&buf, 90),
		)

		if err != nil {
			fmt.Printf("    [!] Failed %s: %v\n", domain, err)
		} else {
			filename := strings.ReplaceAll(domain, ".", "_") + ".png"
			os.WriteFile(filepath.Join(dir, filename), buf, 0644)
		}
		timeoutCancel()
	}
}
