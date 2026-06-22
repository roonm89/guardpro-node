package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

// Constantes oficiales de Guard Pro 7
const TotalSupply = 100000000
const Ticker = "GuardPro"

// Block representa un bloque con Doble SHA-256 y datos firmados
type Block struct {
	Index        int
	Timestamp    string
	Data         string
	PrevHash     string
	Hash         string
	FirmaR       *big.Int // Componente R de la firma ECDSA
	FirmaS       *big.Int // Componente S de la firma ECDSA
}

// CalcularDobleHash aplica SHA-256 dos veces para máxima seguridad de enlace
func CalcularDobleHash(b Block) string {
	record := string(rune(b.Index)) + b.Timestamp + b.Data + b.PrevHash
	
	 primerHash := sha256.Sum256([]byte(record))
	dobleHash := sha256.Sum256(primerHash[:])
	
	return hex.EncodeToString(dobleHash[:])
}

// CrearBloqueFase2 genera un bloque y lo firma criptográficamente con la clave privada
func CrearBloqueFase2(oldBlock Block, data string, privKey *ecdsa.PrivateKey) (Block, error) {
	var newBlock Block
	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = time.Now().String()
	newBlock.Data = data
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = CalcularDobleHash(newBlock)

	// Firmar digitalmente el Hash del bloque para demostrar la identidad del nodo
	hashBytes, _ := hex.DecodeString(newBlock.Hash)
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hashBytes)
	if err != nil {
		return Block{}, err
	}
	newBlock.FirmaR = r
	newBlock.FirmaS = s

	return newBlock, nil
}

func main() {
	fmt.Printf("--- [Fase 2] Iniciando Blockchain Segura de Guard Pro 7 (%s) ---\n", Ticker)
	fmt.Printf("Suministro Total Protegido: %d monedas\n\n", TotalSupply)

	// 1. Generar la Billetera del Nodo (Clave Privada y Pública) usando Curva Elíptica
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println("Error al generar las claves criptográficas:", err)
		return
	}
	pubKeyHash := sha256.Sum256(elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y))
	direccionBilletera := "GP" + hex.EncodeToString(pubKeyHash[:16]) // Dirección simulada de GuardPro

	fmt.Printf("[Billetera del Nodo Creada]\nDirección Pública: %s\n\n", direccionBilletera)

	// 2. Creación del Bloque Génesis (Bloque 0)
	genesisBlock := Block{0, time.Now().String(), "Bloque Genesis Protegido de GuardPro", "", "", nil, nil}
	genesisBlock.Hash = CalcularDobleHash(genesisBlock)
	fmt.Println("[Bloque 0 - Génesis creado con Doble SHA-256]")
	fmt.Printf("Hash Base: %s\n\n", genesisBlock.Hash)

	// 3. Crear Bloque 1 firmado (Simulación de Recompensa por estar activo)
	dataRecompensa := fmt.Sprintf("Recompensa por Uptime asignada a la dirección %s", direccionBilletera)
	bloque1, err := CrearBloqueFase2(genesisBlock, dataRecompensa, privateKey)
	if err != nil {
		fmt.Println("Error al firmar el bloque:", err)
		return
	}

	fmt.Printf("[Bloque %d Creado y Firmado Exitosamente]\n", bloque1.Index)
	fmt.Printf("Data: %s\n", bloque1.Data)
	fmt.Printf("Doble Hash: %s\n", bloque1.Hash)
	fmt.Printf("Firma Digital (R): %s...\n\n", bloque1.FirmaR.Text(16)[:20])

	// 4. Verificación híbrida: El nodo valida que la firma sea real usando la clave pública
	hashBytes, _ := hex.DecodeString(bloque1.Hash)
	esValido := ecdsa.Verify(&privateKey.PublicKey, hashBytes, bloque1.FirmaR, bloque1.FirmaS)
	
	if esValido {
		fmt.Println("✅ [Verificación Exitosa]: La firma es auténtica. El bloque ha sido enlazado a la red.")
	} else {
		fmt.Println("❌ [ALERTA DE SEGURIDAD]: Firma inválida. Intento de alteración detectado.")
	}
}
