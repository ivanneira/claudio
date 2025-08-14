package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DEFAULT_SSH_PORT    = 22
	DEFAULT_TUNNEL_PORT = 2222
	SCRIPTS_DIR        = "./scripts"
)

type Config struct {
	TelegramBotToken string
	TelegramChatID   string
	SSHPort          int
	TunnelPort       int
	TunnelService    string
	LastUpdateID     int64
}

type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		MessageID int64 `json:"message_id"`
		From      struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Date int64  `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

type TelegramResponse struct {
	OK     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

func main() {
	config := loadConfig()
	
	// Verificar configuración de Telegram
	if config.TelegramBotToken == "" || config.TelegramChatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN y TELEGRAM_CHAT_ID son requeridos")
	}

	// Crear directorio de scripts si no existe
	createScriptsDir()

	// Verificar que SSH esté corriendo
	if !isSSHRunning(config.SSHPort) {
		msg := fmt.Sprintf("❌ SSH no está corriendo en puerto %d", config.SSHPort)
		log.Println(msg)
		sendTelegramMessage(config, msg)
		return
	}

	// Crear túnel SSH
	tunnelURL, err := createTunnel(config)
	if err != nil {
		msg := fmt.Sprintf("❌ Error creando túnel: %v", err)
		log.Println(msg)
		sendTelegramMessage(config, msg)
		return
	}

	// Extraer hostname y puerto de la URL para generar comando SSH
	sshCommand := generateSSHCommand(tunnelURL)
	
	// Enviar notificación de éxito
	successMsg := fmt.Sprintf("✅ Túnel SSH activo!\n🌐 URL: %s\n🔌 Puerto local: %d\n\n🔑 Comando SSH:\n```\n%s\n```\n\n💡 Envía !<script> para ejecutar scripts", tunnelURL, config.SSHPort, sshCommand)
	log.Println(successMsg)
	sendTelegramMessage(config, successMsg)

	// Iniciar polling de mensajes de Telegram
	go startTelegramPolling(config)

	// Mantener el programa corriendo
	fmt.Println("Túnel activo. Escuchando comandos de Telegram...")
	fmt.Println("Presiona Ctrl+C para detener")
	select {}
}

func loadConfig() Config {
	// Cargar archivo .env si existe
	loadEnvFile(".env")
	
	return Config{
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
		SSHPort:          getEnvInt("SSH_PORT", DEFAULT_SSH_PORT),
		TunnelPort:       getEnvInt("TUNNEL_PORT", DEFAULT_TUNNEL_PORT),
		TunnelService:    getEnv("TUNNEL_SERVICE", "ngrok"),
		LastUpdateID:     0,
	}
}

func createScriptsDir() {
	if _, err := os.Stat(SCRIPTS_DIR); os.IsNotExist(err) {
		os.MkdirAll(SCRIPTS_DIR, 0755)
		log.Printf("Creado directorio: %s", SCRIPTS_DIR)
	}
}

func startTelegramPolling(config Config) {
	for {
		updates, err := getTelegramUpdates(config)
		if err != nil {
			log.Printf("Error obteniendo updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID > config.LastUpdateID {
				config.LastUpdateID = update.UpdateID
				processMessage(config, update)
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func getTelegramUpdates(config Config) ([]TelegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d", 
		config.TelegramBotToken, config.LastUpdateID+1)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var telegramResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return nil, err
	}

	return telegramResp.Result, nil
}

func processMessage(config Config, update TelegramUpdate) {
	// Verificar que el mensaje venga del chat autorizado
	if strconv.FormatInt(update.Message.Chat.ID, 10) != config.TelegramChatID {
		return
	}

	text := strings.TrimSpace(update.Message.Text)
	
	// Procesar comandos que empiecen con !
	if strings.HasPrefix(text, "!") {
		scriptName := strings.TrimPrefix(text, "!")
		executeScript(config, scriptName, update.Message.From.Username)
	}
}

func executeScript(config Config, scriptName, username string) {
	scriptPath := filepath.Join(SCRIPTS_DIR, scriptName+".sh")
	
	// Verificar que el script existe
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		msg := fmt.Sprintf("❌ Script '%s' no encontrado\n📁 Scripts disponibles: %s", 
			scriptName, getAvailableScripts())
		sendTelegramMessage(config, msg)
		return
	}

	// Enviar notificación de inicio
	startMsg := fmt.Sprintf("🚀 Ejecutando script '%s' solicitado por @%s...", scriptName, username)
	sendTelegramMessage(config, startMsg)
	log.Println(startMsg)

	// Ejecutar script
	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		errorMsg := fmt.Sprintf("❌ Error ejecutando '%s':\n```\n%s\n```", scriptName, string(output))
		sendTelegramMessage(config, errorMsg)
		log.Printf("Error ejecutando script %s: %v", scriptName, err)
		return
	}

	// Enviar resultado exitoso
	successMsg := fmt.Sprintf("✅ Script '%s' ejecutado exitosamente\n```\n%s\n```", scriptName, string(output))
	if len(successMsg) > 4000 { // Límite de Telegram
		successMsg = fmt.Sprintf("✅ Script '%s' ejecutado exitosamente\n📄 Output muy largo, mostrando últimas líneas:\n```\n%s\n```", 
			scriptName, getLastLines(string(output), 20))
	}
	sendTelegramMessage(config, successMsg)
	log.Printf("Script %s ejecutado exitosamente", scriptName)
}

func getAvailableScripts() string {
	files, err := os.ReadDir(SCRIPTS_DIR)
	if err != nil {
		return "Error leyendo directorio de scripts"
	}

	var scripts []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sh") {
			scriptName := strings.TrimSuffix(file.Name(), ".sh")
			scripts = append(scripts, scriptName)
		}
	}

	if len(scripts) == 0 {
		return "Ninguno"
	}

	return strings.Join(scripts, ", ")
}

func getLastLines(text string, lines int) string {
	allLines := strings.Split(text, "\n")
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}
	return strings.Join(allLines[start:], "\n")
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// Archivo .env no existe, continuar con variables de entorno del sistema
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Ignorar líneas vacías y comentarios
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parsear líneas en formato KEY=VALUE
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				// Solo establecer si la variable no está ya definida en el entorno
				if os.Getenv(key) == "" {
					os.Setenv(key, value)
				}
			}
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func isSSHRunning(port int) bool {
	cmd := exec.Command("ss", "-ln")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	portStr := fmt.Sprintf(":%d", port)
	return strings.Contains(string(output), portStr)
}

func createTunnel(config Config) (string, error) {
	switch config.TunnelService {
	case "ngrok":
		return createNgrokTunnel(config.SSHPort)
	case "localtunnel":
		return createLocalTunnel(config.TunnelPort)
	default:
		return "", fmt.Errorf("servicio de túnel no soportado: %s", config.TunnelService)
	}
}

func createNgrokTunnel(sshPort int) (string, error) {
	if _, err := exec.LookPath("ngrok"); err != nil {
		return "", fmt.Errorf("ngrok no está instalado")
	}

	// Configurar authtoken si está disponible
	if authtoken := getEnv("NGROK_AUTHTOKEN", ""); authtoken != "" {
		cmd := exec.Command("ngrok", "authtoken", authtoken)
		if err := cmd.Run(); err != nil {
			log.Printf("Advertencia: Error configurando authtoken de ngrok: %v", err)
		}
	}

	cmd := exec.Command("ngrok", "tcp", strconv.Itoa(sshPort))
	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("error iniciando ngrok: %v", err)
	}

	// Esperar a que ngrok se inicie
	time.Sleep(5 * time.Second)

	// Obtener la URL real del API de ngrok
	resp, err := http.Get("http://localhost:4040/api/tunnels")
	if err != nil {
		return "", fmt.Errorf("error obteniendo URL de ngrok: %v", err)
	}
	defer resp.Body.Close()

	var ngrokResponse struct {
		Tunnels []struct {
			PublicURL string `json:"public_url"`
			Proto     string `json:"proto"`
		} `json:"tunnels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ngrokResponse); err != nil {
		return "", fmt.Errorf("error decodificando respuesta de ngrok: %v", err)
	}

	if len(ngrokResponse.Tunnels) == 0 {
		return "", fmt.Errorf("no se encontraron túneles activos en ngrok")
	}

	// Buscar el túnel TCP
	for _, tunnel := range ngrokResponse.Tunnels {
		if tunnel.Proto == "tcp" {
			return tunnel.PublicURL, nil
		}
	}

	return "", fmt.Errorf("no se encontró túnel TCP activo")
}

func createLocalTunnel(port int) (string, error) {
	if _, err := exec.LookPath("lt"); err != nil {
		return "", fmt.Errorf("localtunnel no está instalado")
	}

	cmd := exec.Command("lt", "--port", strconv.Itoa(port))
	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("error iniciando localtunnel: %v", err)
	}

	return fmt.Sprintf("https://random-subdomain.loca.lt:%d", port), nil
}

func sendTelegramMessage(config Config, message string) {
	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramBotToken)
	
	data := url.Values{}
	data.Set("chat_id", config.TelegramChatID)
	data.Set("text", message)
	data.Set("parse_mode", "Markdown")

	resp, err := http.PostForm(telegramURL, data)
	if err != nil {
		log.Printf("Error enviando mensaje a Telegram: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Error en respuesta de Telegram: %d - %s", resp.StatusCode, string(body))
	}
}

func generateSSHCommand(tunnelURL string) string {
	// Parsear URL del túnel (formato: tcp://hostname:puerto)
	parsedURL, err := url.Parse(tunnelURL)
	if err != nil {
		return fmt.Sprintf("ssh ineira@%s", tunnelURL)
	}
	
	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	
	if port == "" {
		return fmt.Sprintf("ssh ineira@%s", hostname)
	}
	
	return fmt.Sprintf("ssh ineira@%s -p %s", hostname, port)
}