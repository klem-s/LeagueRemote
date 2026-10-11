// LoL Remote - choisis ton champion, tes runes et suis la draft depuis ton téléphone.
//
// Usage : lol-remote.exe
// Puis ouvre l'adresse affichée sur ton téléphone (même Wi-Fi que le PC).
// Un seul exécutable, aucune installation.
//
// This is the composition root: it is the only place that wires the
// concrete lcu.Client (infrastructure) into league.Service (use cases)
// and httpapi.Server (interface adapter). Nothing downstream imports
// upward.
package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"

	"lolremote/internal/httpapi"
	"lolremote/internal/lcu"
	"lolremote/internal/league"
)

const port = 8080

func localIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func main() {
	pin := os.Getenv("LOL_REMOTE_PIN")
	if pin == "" {
		pin = strconv.Itoa(1000 + rand.Intn(9000))
	}

	svc := league.NewService(lcu.New())
	go svc.Watch()

	fmt.Println()
	fmt.Println("  LoL Remote lancé")
	fmt.Printf("  Sur ton téléphone : http://%s:%d\n", localIP(), port)
	fmt.Printf("  Code PIN          : %s\n\n", pin)
	fmt.Println("  (Ctrl+C pour arrêter)")
	fmt.Println()

	server := httpapi.NewServer(svc, pin)
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		fmt.Println("Erreur serveur:", err)
		os.Exit(1)
	}
}
