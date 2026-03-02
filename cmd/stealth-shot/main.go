package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"golang.org/x/time/rate"
)

func main() {
	inputFile := flag.String("f", "", "File containing subdomains (required)")
	outputDir := flag.String("o", "results", "Output directory")
	threads := flag.Int("t", 3, "Number of concurrent threads")
	delay := flag.Int("d", 5, "Delay in seconds between requests")
	proxy := flag.String("proxy", "", "Optional: Proxy server URL")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("\nUsage: stealth-shot -f hosts.txt -o results -t 3 -d 5")
		os.Exit(1)
	}

	os.MkdirAll(*outputDir, 0755)
	
	// Initialize Files
	csvFile, _ := os.Create(filepath.Join(*outputDir, "summary.csv"))
	defer csvFile.Close()
	mdFile, _ := os.Create(filepath.Join(*outputDir, "report.md"))
	defer mdFile.Close()

	// Writers
	csvWriter := csv.NewWriter(csvFile)
	csvWriter.Write([]string{"URL", "Status", "Title", "Screenshot"})
	
	fmt.Fprintln(mdFile, "# Stealth-Shot Recon Report")
	fmt.Fprintln(mdFile, "| URL | Status | Title | Screenshot |")
	fmt.Fprintln(mdFile, "| :--- | :--- | :--- | :--- |")

	var mu sync.Mutex // Protects concurrent writes to files
	limiter := rate.NewLimiter(rate.Every(time.Duration(*delay)*time.Second), 1)
	
	jobs := make(chan string)
	var wg sync.WaitGroup

	fmt.Printf("[!] Starting Stealth-Shot | Threads: %d | Delay: %ds\n\n", *threads, *delay)

	for w := 1; w <= *threads; w++ {
		wg.Add(1)
		go worker(w, jobs, &wg, *outputDir, limiter, *proxy, csvWriter, mdFile, &mu)
	}

	file, _ := os.Open(*inputFile)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		jobs <- scanner.Text()
	}
	close(jobs)
	wg.Wait()
	
	csvWriter.Flush()
	fmt.Println("\n[✔] Done! View 'report.md' on GitHub for the visual gallery.")
}

func worker(id int, jobs <-chan string, wg *sync.WaitGroup, dir string, limiter *rate.Limiter, proxyAddr string, csvWriter *csv.Writer, mdFile *os.File, mu *sync.Mutex) {
	defer wg.Done()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	)
	if proxyAddr != "" {
		opts = append(opts, chromedp.ProxyServer(proxyAddr))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	for domain := range jobs {
		limiter.Wait(context.Background())
		targetURL := domain
		if !strings.HasPrefix(targetURL, "http") {
			targetURL = "https://" + targetURL
		}

		ctx, _ := chromedp.NewContext(allocCtx)
		ctx, tCancel := context.WithTimeout(ctx, 45*time.Second)

		var buf []byte
		var title string
		var statusCode int64

		// Listen for Status Code
		chromedp.ListenTarget(ctx, func(ev interface{}) {
			if response, ok := ev.(*network.EventResponseReceived); ok {
				if response.Response.URL == targetURL || response.Response.URL == targetURL+"/" {
					statusCode = response.Response.Status
				}
			}
		})

		err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil).Do(ctx)
			}),
			chromedp.Navigate(targetURL),
			chromedp.Title(&title),
			chromedp.Sleep(10*time.Second),
			chromedp.FullScreenshot(&buf, 90),
		)

		if err == nil {
			filename := strings.ReplaceAll(domain, ".", "_") + ".png"
			os.WriteFile(filepath.Join(dir, filename), buf, 0644)
			
			// Critical Section: Writing to reports
			mu.Lock()
			csvWriter.Write([]string{targetURL, fmt.Sprintf("%d", statusCode), title, filename})
			fmt.Fprintf(mdFile, "| %s | %d | %s | ![Shot](%s) |\n", targetURL, statusCode, title, filename)
			mu.Unlock()
			
			fmt.Printf("[Worker %d] [+] %s [%d]\n", id, targetURL, statusCode)
		} else {
			fmt.Printf("[Worker %d] [!] Failed: %s\n", id, targetURL)
		}
		tCancel()
	}
}
