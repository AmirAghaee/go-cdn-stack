package service

import (
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/repository"
	"github.com/gin-gonic/gin"
)

type EdgeServiceInterface interface {
	Register(c *gin.Context)
}

type edgeService struct {
	edgeRepository repository.EdgeRepositoryInterface
}

func NewEdgeService(edgeRepo repository.EdgeRepositoryInterface) EdgeServiceInterface {
	return &edgeService{
		edgeRepository: edgeRepo,
	}
}

func (s *edgeService) Register(c *gin.Context) {
	var edge domain.Edge
	if err := c.ShouldBindJSON(&edge); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	s.edgeRepository.Set(edge)
	c.JSON(200, gin.H{"status": "registered", "instance": edge.Instance})
}
