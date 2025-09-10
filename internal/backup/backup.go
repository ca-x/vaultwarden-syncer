package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

type Service struct {
	vaultwardenDataPath string
	compressionLevel    int
	password            string
}

type BackupOptions struct {
	VaultwardenDataPath string
	CompressionLevel    int
	Password            string
}

func NewService(opts BackupOptions) *Service {
	return &Service{
		vaultwardenDataPath: opts.VaultwardenDataPath,
		compressionLevel:    opts.CompressionLevel,
		password:            opts.Password,
	}
}

func (s *Service) CreateBackup(ctx context.Context) (io.Reader, string, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("vaultwarden-backup-%s.zip", timestamp)

	var buf bytes.Buffer
	
	if err := s.createZipArchive(&buf); err != nil {
		return nil, "", fmt.Errorf("failed to create zip archive: %w", err)
	}

	data := buf.Bytes()

	if s.password != "" {
		encryptedData, err := s.encryptData(data)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encrypt backup: %w", err)
		}
		filename = strings.Replace(filename, ".zip", ".enc", 1)
		return bytes.NewReader(encryptedData), filename, nil
	}

	return bytes.NewReader(data), filename, nil
}

func (s *Service) createZipArchive(w io.Writer) error {
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	return filepath.Walk(s.vaultwardenDataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(s.vaultwardenDataPath, path)
		if err != nil {
			return err
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		return err
	})
}

func (s *Service) encryptData(data []byte) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key([]byte(s.password), salt, 10000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	
	result := make([]byte, len(salt)+len(ciphertext))
	copy(result, salt)
	copy(result[len(salt):], ciphertext)

	return result, nil
}

func (s *Service) DecryptData(data []byte) ([]byte, error) {
	if s.password == "" {
		return nil, fmt.Errorf("no password set for decryption")
	}

	if len(data) < 32 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	salt := data[:32]
	ciphertext := data[32:]

	key := pbkdf2.Key([]byte(s.password), salt, 10000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *Service) ExtractBackup(ctx context.Context, data io.Reader, destPath string) error {
	zipData, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read backup data: %w", err)
	}

	if s.password != "" && len(zipData) > 32 {
		decryptedData, err := s.DecryptData(zipData)
		if err != nil {
			return fmt.Errorf("failed to decrypt backup: %w", err)
		}
		zipData = decryptedData
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, file := range zipReader.File {
		if strings.Contains(file.Name, "..") {
			continue
		}

		destFile := filepath.Join(destPath, file.Name)
		
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destFile, file.FileInfo().Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(destFile)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}