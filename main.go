// LoL Remote - choisis ton champion, tes runes et suis la draft depuis ton téléphone.
//
// Lance l'exécutable : une icône apparaît dans la zone de notification avec
// l'adresse et le code PIN à saisir sur le téléphone (même Wi-Fi que le PC).
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
	"lolremote/internal/trayui"
)

// Port par défaut : moins commun que 8080, pour limiter les conflits
// avec d'autres outils locaux. Choisi arbitrairement, sans signification.
const defaultPort = 53217

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

	// Port fixe par défaut (moins commun que 8080, donc moins souvent pris
	// par autre chose) ; si déjà occupé, le système en choisit un libre.
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", defaultPort))
	if err != nil {
		listener, err = net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			trayui.ReportFatal(fmt.Sprintf("Impossible d'ouvrir un port : %v", err))
			os.Exit(1)
		}
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://%s:%d", localIP(), port)

	svc := league.NewService(lcu.New())
	go svc.Watch()

	server := httpapi.NewServer(svc, pin)
	go func() {
		if err := http.Serve(listener, server.Handler()); err != nil {
			trayui.ReportFatal(fmt.Sprintf("Erreur serveur : %v", err))
			os.Exit(1)
		}
	}()

	trayui.Run(pin, url) // bloque : boucle de l'icône dans la zone de notification
}
