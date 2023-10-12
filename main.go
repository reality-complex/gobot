package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	"os"
)

const etherscanAPI = "https://api-goerli.etherscan.io/api"

type BlockResponse struct {
	Result BlockResult `json:"result"`
}

type BlockResult struct {
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	To   string `json:"to"`
	Hash string `json:"hash"`
}

type ABIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func getContractABI(apiKey, contractAddress string) (string, error) {
	url := fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s", etherscanAPI, contractAddress, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var abiResponse ABIResponse
	err = json.Unmarshal(body, &abiResponse)
	if err != nil {
		return "", err
	}

	if abiResponse.Status != "1" {
		return "", fmt.Errorf("Failed to fetch ABI: %s", abiResponse.Message)
	}

	return abiResponse.Result, nil
}

func hasPriceFunction(abiStr string) bool {
	// List of common price-related function names
	priceFunctions := []string{
		"getPrice",
		"price",
		"currentPrice",
		"fetchPrice",
		// ... add more as needed
	}

	for _, functionName := range priceFunctions {
		if strings.Contains(abiStr, functionName) {
			return true
		}
	}
	return false
}


func getPendingTransactionsABI(apiKey string) ([]string, error) {
	url := fmt.Sprintf("%s?module=proxy&action=eth_getBlockByNumber&tag=pending&boolean=true&apikey=%s", etherscanAPI, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var blockResponse BlockResponse
	err = json.Unmarshal(body, &blockResponse)
	if err != nil {
		return nil, err
	}

	var abis []string
	for _, tx := range blockResponse.Result.Transactions {
		if tx.To != "" {
			abi, err := getContractABI(apiKey, tx.To)
			if err == nil && !strings.Contains(abi, "Contract source code not verified") {
				abis = append(abis, abi)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	return abis, nil
}

func main() {
	apiKey := os.Getenv("ETHERSCAN_API_KEY")
	for {
		abis, err := getPendingTransactionsABI(apiKey)
		if err != nil {
			log.Printf("Error fetching ABIs: %v", err)
		} else {
			for _, abi := range abis {
				fmt.Println("ABI:", abi)
			}
		}
	}
}
