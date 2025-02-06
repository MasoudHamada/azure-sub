import sys
import requests
from concurrent.futures import ThreadPoolExecutor

# Configuration
WORDLIST = "/usr/share/seclists/Discovery/DNS/subdomains-top1million-110000.txt"
BASE_URLS = [
    "azurewebsites.net",
    "blob.core.windows.net",
    "queue.core.windows.net",
    "file.core.windows.net",
    "table.core.windows.net",
    "scm.azurewebsites.net"
]
PERMUTATIONS = [
    "{word}-{company}", "{company}-{word}",
    "{word}{company}", "{company}{word}"
]

# Function to check if a subdomain exists
def check_subdomain(subdomain):
    url = f"https://{subdomain}"
    try:
        response = requests.get(url, timeout=5)
        if response.status_code in [200, 302]:  # Detect valid subdomains
            print(f"[VALID] {subdomain} ({response.status_code})")
            with open("valid_subdomains.txt", "a") as f:
                f.write(subdomain + "\n")
    except requests.exceptions.RequestException:
        pass  # Ignore unreachable domains

# Function to generate and test subdomains
def enumerate_subdomains(company):
    with open(WORDLIST, "r") as file:
        words = [line.strip() for line in file]

    subdomains = []
    for base in BASE_URLS:
        for word in words:
            for pattern in PERMUTATIONS:
                subdomain = pattern.format(company=company, word=word) + "." + base
                subdomains.append(subdomain)

    print(f"[*] Testing {len(subdomains)} subdomains...")

    # Use ThreadPoolExecutor to speed up enumeration
    with ThreadPoolExecutor(max_workers=50) as executor:
        executor.map(check_subdomain, subdomains)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python3 azure_enum.py <company_name>")
        sys.exit(1)
    
    enumerate_subdomains(sys.argv[1])
