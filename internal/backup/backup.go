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

	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"
)

type Service struct {
	vaultwardenDataPath string
	compressionLevel    int
	password            string
	logger              *zap.Logger
}

type BackupOptions struct {
	VaultwardenDataPath string
	CompressionLevel    int
	Password            string
	Logger              *zap.Logger
}

func NewService(opts BackupOptions) *Service {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop() // Use no-op logger if none provided
	}
	return &Service{
		vaultwardenDataPath: opts.VaultwardenDataPath,
		compressionLevel:    opts.CompressionLevel,
		password:            opts.Password,
		logger:              logger,
	}
}

func (s *Service) CreateBackup(ctx context.Context) (io.Reader, string, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("vaultwarden-backup-%s.zip", timestamp)

	s.logger.Info("Starting backup creation", zap.String("filename", filename))

	// Check if data path exists
	if _, err := os.Stat(s.vaultwardenDataPath); os.IsNotExist(err) {
		s.logger.Error("Vaultwarden data path does not exist", zap.String("path", s.vaultwardenDataPath))
		return nil, "", fmt.Errorf("vaultwarden data path does not exist: %s", s.vaultwardenDataPath)
	}

	var buf bytes.Buffer

	if err := s.createZipArchive(&buf); err != nil {
		s.logger.Error("Failed to create zip archive", zap.Error(err))
		return nil, "", fmt.Errorf("failed to create zip archive: %w", err)
	}

	data := buf.Bytes()
	s.logger.Info("Backup archive created successfully", zap.Int("size_bytes", len(data)))

	if s.password != "" {
		s.logger.Info("Encrypting backup with password")
		encryptedData, err := s.encryptData(data)
		if err != nil {
			s.logger.Error("Failed to encrypt backup", zap.Error(err))
			return nil, "", fmt.Errorf("failed to encrypt backup: %w", err)
		}
		filename = strings.Replace(filename, ".zip", ".enc", 1)
		s.logger.Info("Backup encrypted successfully", zap.String("encrypted_filename", filename))
		return bytes.NewReader(encryptedData), filename, nil
	}

	s.logger.Info("Backup created successfully", zap.String("filename", filename))
	return bytes.NewReader(data), filename, nil
}

func (s *Service) createZipArchive(w io.Writer) error {
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	s.logger.Debug("Creating zip archive from path", zap.String("path", s.vaultwardenDataPath))
	filesAdded := 0

	err := filepath.Walk(s.vaultwardenDataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Warn("Error accessing file during backup", zap.String("path", path), zap.Error(err))
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(s.vaultwardenDataPath, path)
		if err != nil {
			s.logger.Error("Failed to get relative path", zap.String("path", path), zap.Error(err))
			return err
		}

		s.logger.Debug("Adding file to archive", zap.String("relative_path", relPath))

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			s.logger.Error("Failed to create zip file entry", zap.String("relative_path", relPath), zap.Error(err))
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			s.logger.Error("Failed to open file for reading", zap.String("path", path), zap.Error(err))
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		if err != nil {
			s.logger.Error("Failed to copy file to archive", zap.String("path", path), zap.Error(err))
			return err
		}

		filesAdded++
		return nil
	})

	if err != nil {
		return err
	}

	s.logger.Info("Archive creation completed", zap.Int("files_added", filesAdded))
	return nil
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

// CalculateChecksum 计算备份数据的校验和以避免重复备份
func (s *Service) CalculateChecksum(reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read data for checksum: %w", err)
	}

	// 计算SHA256校验和
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// GetDataInfo 获取数据目录的信息用于比较
func (s *Service) GetDataInfo() (map[string]time.Time, error) {
	info := make(map[string]time.Time)

	err := filepath.Walk(s.vaultwardenDataPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fileInfo.IsDir() {
			relPath, err := filepath.Rel(s.vaultwardenDataPath, path)
			if err != nil {
				return err
			}
			info[relPath] = fileInfo.ModTime()
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk data directory: %w", err)
	}

	return info, nil
}
