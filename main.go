package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
)

type IPInfo struct {
	IP      string
	Country string
	City    string
	Region  string
	ASN     string
	Org     string
	ASNOrg  string
	Error   string
}

type DomainResult struct {
	DomainQuery string
	IpApiIs     []IPInfo
	IpApiCom    []IPInfo
}

func fetchIpApiIs(ip string) (IPInfo, error) {
	var info IPInfo
	url := fmt.Sprintf("https://api.ipapi.is/?q=%s", ip)
	resp, err := http.Get(url)
	if err != nil {
		return info, fmt.Errorf("erreur lors de la requête ipapi.is: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("ipapi.is a retourné un code %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return info, fmt.Errorf("erreur de décodage JSON ipapi.is: %v", err)
	}

	info.IP = ip
	if location, ok := result["location"].(map[string]interface{}); ok {
		if val, ok := location["country"].(string); ok {
			info.Country = val
		}
		if val, ok := location["city"].(string); ok {
			info.City = val
		}
		if val, ok := location["state"].(string); ok {
			info.Region = val
		}
	}

	if asn, ok := result["asn"].(map[string]interface{}); ok {
		if val, ok := asn["asn"].(float64); ok {
			info.ASN = fmt.Sprintf("AS%.0f", val)
		}
		if val, ok := asn["org"].(string); ok {
			info.Org = val
		}
		if val, ok := asn["descr"].(string); ok {
			info.ASNOrg = val
		}
	}

	if company, ok := result["company"].(map[string]interface{}); ok {
		if val, ok := company["domain"].(string); ok {
			info.ASNOrg = val
		}
	}

	return info, nil
}

func fetchIpApiCom(ip string) (IPInfo, error) {
	var info IPInfo
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	resp, err := http.Get(url)
	if err != nil {
		return info, fmt.Errorf("erreur lors de la requête ip-api.com: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("ip-api.com a retourné un code %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return info, fmt.Errorf("erreur de décodage JSON ip-api.com: %v", err)
	}

	if status, ok := result["status"].(string); !ok || status != "success" {
		return info, fmt.Errorf("requête ip-api.com échouée: %v", result["message"])
	}

	info.IP = ip
	if val, ok := result["country"].(string); ok {
		info.Country = val
	}
	if val, ok := result["city"].(string); ok {
		info.City = val
	}
	if val, ok := result["regionName"].(string); ok {
		info.Region = val
	}
	if val, ok := result["isp"].(string); ok {
		info.Org = val
	}
	if val, ok := result["org"].(string); ok {
		info.ASNOrg = val
	}
	if val, ok := result["as"].(string); ok {
		info.ASN = val
	}

	return info, nil
}

func lookupHandler(w http.ResponseWriter, r *http.Request) {
	data := DomainResult{
		DomainQuery: "", // Toujours initialisé
	}

	if r.Method == http.MethodPost {
		domain := r.FormValue("domain")
		if domain == "" {
			http.Error(w, "Veuillez entrer un domaine", http.StatusBadRequest)
			return
		}
		data.DomainQuery = domain

		ips, err := net.LookupIP(domain)
		if err != nil {
			data.IpApiIs = append(data.IpApiIs, IPInfo{
				IP:    "N/A",
				Error: fmt.Sprintf("Impossible de résoudre le domaine: %v", err),
			})
			data.IpApiCom = append(data.IpApiCom, IPInfo{
				IP:    "N/A",
				Error: fmt.Sprintf("Impossible de résoudre le domaine: %v", err),
			})
		} else {
			for _, ip := range ips {
				ipStr := ip.String()

				// ipapi.is
				infoIs, err := fetchIpApiIs(ipStr)
				if err != nil {
					infoIs.Error = err.Error()
					infoIs.IP = ipStr
				}
				data.IpApiIs = append(data.IpApiIs, infoIs)

				// ip-api.com
				infoCom, err := fetchIpApiCom(ipStr)
				if err != nil {
					infoCom.Error = err.Error()
					infoCom.IP = ipStr
				}
				data.IpApiCom = append(data.IpApiCom, infoCom)
			}
		}
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Erreur de template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func main() {
	http.HandleFunc("/", lookupHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Serveur démarré sur http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
