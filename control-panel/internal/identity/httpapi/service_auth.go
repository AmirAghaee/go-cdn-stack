package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServiceAuth authenticates machine callers with a credential separate from human JWTs.
// Route registration determines the credential's scope.
func ServiceAuth(expectedToken string) gin.HandlerFunc {
	expectedDigest := sha256.Sum256([]byte(expectedToken))
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		valid := len(parts) == 2 && parts[0] == "Bearer" && expectedToken != ""
		if valid {
			actualDigest := sha256.Sum256([]byte(parts[1]))
			valid = subtle.ConstantTimeCompare(actualDigest[:], expectedDigest[:]) == 1
		}
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid service credentials"})
			c.Abort()
			return
		}
		c.Next()
	}
}
