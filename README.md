# Stealth-Shot 📸

**Stealth-Shot** is a high-performance, WAF-evading subdomain screenshotter built in Go. It uses the Chrome DevTools Protocol (CDP) to mimic human browsing behavior, bypassing common "headless" detection used by enterprise WAFs like Akamai and Cloudflare.



## ⚡ Installation

Ensure you have [Go 1.21+](https://go.dev/doc/install) installed. Then, run the following command to install the binary to your `$GOPATH/bin`:

```bash
go install github.com/Gopi-eng2202/stealth-shot/cmd/stealth-shot@latest
```
## 🛠️ Usage
Once installed, you can run stealth-shot from anywhere in your terminal.

Basic Scan
Scan a list of subdomains and save screenshots to a default folder:
```
stealth-shot -f hosts.txt -o results
```
Advanced Stealth Scan (Recommended for Akamai)
Use fewer threads and a longer delay to stay under the radar:
```
stealth-shot -f hosts.txt -o universal_recon -t 2 -d 15
```
Proxy Support (Tor/Burp)
Route your traffic through a SOCKS5 or HTTP proxy:
```
stealth-shot -f hosts.txt -o proxy_scan -proxy socks5://127.0.0.1:9050
```
## ⚙️ Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| **`-f`** | **(Required)** Path to the file containing subdomains/URLs. | `""` |
| **`-o`** | Output directory where screenshots will be saved. | `results` |
| **`-t`** | Number of concurrent browser threads (Goroutines). | `3` |
| **`-d`** | Delay in seconds between requests per thread. | `5` |
| **`-proxy`** | Proxy server URL (e.g., `socks5://127.0.0.1:9050`). | `""` |

## 🕵️ Why Stealth-Shot?
Most automated screenshotting tools are blocked by modern WAFs because they use default headless signatures.
Stealth-Shot implements:

- AutomationControlled Flag Removal: Native Chrome evasion.

- navigator.webdriver Spoofing: Injects JS to hide automation properties.

- Worker Pool Pattern: Efficiently manages resources without "bursting" the WAF.

## 🌟 Features

- **Stealth Engine:** Bypasses `navigator.webdriver` detection.
- **Worker Pool:** Thread-safe concurrent scanning.
- **Rate Limiting:** Staggered requests to evade WAF burst detection.
- **Metadata Extraction:** (NEW) Captures HTTP Status Codes and Page Titles.
- **CSV Reporting:** (NEW) Generates a `summary.csv` for easy analysis.
