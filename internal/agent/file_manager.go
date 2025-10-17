package agent

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"remoteshell/internal/common"
)

// FileManager gère le transfert de fichiers
type FileManager struct {
	basePath  string
	chunkSize int
}

// NewFileManager crée un nouveau gestionnaire de fichiers
func NewFileManager(basePath string, chunkSize int) *FileManager {
	if basePath == "" {
		basePath = "."
	}

	// Créer le répertoire de base s'il n'existe pas
	if err := os.MkdirAll(basePath, 0755); err != nil {
		// Fallback vers le répertoire courant
		basePath = "."
	}

	return &FileManager{
		basePath:  basePath,
		chunkSize: chunkSize,
	}
}

// ListFiles liste les fichiers d'un répertoire
func (fm *FileManager) ListFiles(path string) ([]*common.FileData, error) {
	fullPath := fm.getFullPath(path)

	log.Printf("DEBUG: ListFiles - chemin demandé: %s", path)
	log.Printf("DEBUG: ListFiles - chemin complet: %s", fullPath)
	log.Printf("DEBUG: ListFiles - répertoire de base: %s", fm.basePath)

	// Vérifier que le chemin est dans le répertoire de base
	if !fm.isPathSafe(fullPath) {
		log.Printf("DEBUG: ListFiles - chemin non autorisé: %s", fullPath)
		return nil, fmt.Errorf("chemin non autorisé")
	}

	log.Printf("DEBUG: ListFiles - chemin autorisé, accès au répertoire: %s", fullPath)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		// C'est un fichier, retourner ses informations
		return []*common.FileData{fm.fileInfoToFileData(stat, path)}, nil
	}

	// C'est un répertoire, lister son contenu
	entries, err := file.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	var files []*common.FileData
	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		entryFullPath := filepath.Join(fullPath, entry.Name())

		entryStat, err := os.Stat(entryFullPath)
		if err != nil {
			continue
		}

		fileData := fm.fileInfoToFileData(entryStat, entryPath)
		files = append(files, fileData)
	}

	return files, nil
}

// UploadFile gère l'upload d'un fichier
func (fm *FileManager) UploadFile(path string, chunks []*common.FileChunk) error {
	fullPath := fm.getFullPath(path)

	// Vérifier que le chemin est dans le répertoire de base
	if !fm.isPathSafe(fullPath) {
		return fmt.Errorf("chemin non autorisé")
	}

	// Créer le répertoire parent s'il n'existe pas
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire: %v", err)
	}

	// Ouvrir le fichier en écriture
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier: %v", err)
	}
	defer file.Close()

	// Écrire les chunks dans l'ordre
	for _, chunk := range chunks {
		if _, err := file.WriteAt(chunk.Data, chunk.Offset); err != nil {
			return fmt.Errorf("erreur d'écriture du chunk: %v", err)
		}
	}

	return nil
}

// DownloadFile gère le téléchargement d'un fichier
func (fm *FileManager) DownloadFile(path string) ([]*common.FileChunk, error) {
	fullPath := fm.getFullPath(path)

	// Vérifier que le chemin est dans le répertoire de base
	if !fm.isPathSafe(fullPath) {
		return nil, fmt.Errorf("chemin non autorisé")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("ne peut pas télécharger un répertoire")
	}

	var chunks []*common.FileChunk
	buffer := make([]byte, fm.chunkSize)
	offset := int64(0)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("erreur de lecture: %v", err)
		}

		if n == 0 {
			break
		}

		chunk := &common.FileChunk{
			Path:   path,
			Offset: offset,
			Data:   make([]byte, n),
			IsLast: false,
		}

		copy(chunk.Data, buffer[:n])

		// Calculer le checksum
		hash := md5.Sum(chunk.Data)
		chunk.Checksum = hex.EncodeToString(hash[:])

		chunks = append(chunks, chunk)
		offset += int64(n)

		if err == io.EOF {
			chunk.IsLast = true
			break
		}
	}

	return chunks, nil
}

// DeleteFile supprime un fichier ou un répertoire
func (fm *FileManager) DeleteFile(path string) error {
	fullPath := fm.getFullPath(path)

	// Vérifier que le chemin est dans le répertoire de base
	if !fm.isPathSafe(fullPath) {
		return fmt.Errorf("chemin non autorisé")
	}

	return os.RemoveAll(fullPath)
}

// CreateDirectory crée un répertoire
func (fm *FileManager) CreateDirectory(path string) error {
	fullPath := fm.getFullPath(path)

	// Vérifier que le chemin est dans le répertoire de base
	if !fm.isPathSafe(fullPath) {
		return fmt.Errorf("chemin non autorisé")
	}

	return os.MkdirAll(fullPath, 0755)
}

// GetFileInfo retourne les informations d'un fichier
func (fm *FileManager) GetFileInfo(path string) (*common.FileData, error) {
	fullPath := fm.getFullPath(path)

	// Vérifier que le chemin est dans le répertoire de base
	if !fm.isPathSafe(fullPath) {
		return nil, fmt.Errorf("chemin non autorisé")
	}

	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return fm.fileInfoToFileData(stat, path), nil
}

// getFullPath retourne le chemin complet d'un fichier
func (fm *FileManager) getFullPath(path string) string {
	// Nettoyer le chemin
	cleanPath := filepath.Clean(path)

	// Si le chemin est absolu, le retourner tel quel
	if filepath.IsAbs(cleanPath) {
		return cleanPath
	}

	// Mode root : pour les chemins relatifs, commencer depuis la racine
	// Cela permet d'accéder à tous les répertoires du système
	return filepath.Join("/", cleanPath)
}

// isPathSafe vérifie qu'un chemin est dans le répertoire de base
func (fm *FileManager) isPathSafe(path string) bool {
	// Mode root : autoriser tous les chemins
	// En production, vous pourriez vouloir réactiver les restrictions de sécurité
	return true

	// Code original (désactivé pour l'accès root) :
	// Nettoyer les chemins
	// cleanPath := filepath.Clean(path)
	// cleanBase := filepath.Clean(fm.basePath)
	//
	// // Vérifier que le chemin commence par le répertoire de base
	// rel, err := filepath.Rel(cleanBase, cleanPath)
	// if err != nil {
	// 	return false
	// }
	//
	// // Vérifier qu'il n'y a pas de ".." dans le chemin relatif
	// return !strings.Contains(rel, "..")
}

// fileInfoToFileData convertit os.FileInfo en FileData
func (fm *FileManager) fileInfoToFileData(stat os.FileInfo, path string) *common.FileData {
	return &common.FileData{
		Path:     path,
		Size:     stat.Size(),
		Mode:     uint32(stat.Mode()),
		Modified: stat.ModTime(),
		IsDir:    stat.IsDir(),
	}
}

// GetBasePath retourne le répertoire de base
func (fm *FileManager) GetBasePath() string {
	return fm.basePath
}

// SetBasePath définit un nouveau répertoire de base
func (fm *FileManager) SetBasePath(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	fm.basePath = path
	return nil
}
