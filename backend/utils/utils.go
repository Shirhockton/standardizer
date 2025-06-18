package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	return string(hash), err

}

func GenerateJWT(userName string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": userName,
		"exp":      jwt.TimeFunc().Add(15 * 60 * time.Minute).Unix(),
	})
	sigendToken, err := token.SignedString([]byte("secret"))
	return "Bearer " + sigendToken, err
}

func CheckPassword(pwd, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	return err == nil
}

func ParseJWT(tokenString string) (string, error) {
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte("secret"), nil
		})
		if err != nil {
			return "", err
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			username, ok := claims["username"].(string)
			if !ok {
				return "", jwt.ErrSignatureInvalid
			}
			return username, nil
		} else {
			return "", jwt.ErrSignatureInvalid
		}

	} else {
		return "", jwt.ErrSignatureInvalid
	}

}

// 辅助函数：分块代码
func SplitCodeIntoChunks(code string, linesPerChunk int) []string {
	lines := strings.Split(code, "\n")
	var chunks []string

	for i := 0; i < len(lines); i += linesPerChunk {
		end := i + linesPerChunk
		if end > len(lines) {
			end = len(lines)
		}
		chunks = append(chunks, strings.Join(lines[i:end], "\n"))
	}
	return chunks
}

// 辅助函数：字符串转整数
func Atoi(s string) int {
	i := 0
	for _, r := range s {
		i = i*10 + int(r-'0')
	}
	return i
}

func CalcMd5(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	md5Hash := md5.Sum(content)
	md5Low32 := hex.EncodeToString(md5Hash[:])[:32]
	return md5Low32
}
