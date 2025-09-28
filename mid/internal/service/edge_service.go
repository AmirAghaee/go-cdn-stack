package service

import (
	"fmt"
	"net/http"

	"github.com/AmirAghaee/go-cdn-stack/mid/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/mid/internal/repository"
	"github.com/gin-gonic/gin"
)

type EdgeServiceInterface interface {
	Register(c *gin.Context)
	GetCdns(c *gin.Context)
}

type edgeService struct {
	edgeRepository repository.EdgeRepositoryInterface
	cdnRepository  repository.CdnRepositoryInterface
}

func NewEdgeService(edgeRepo repository.EdgeRepositoryInterface, cdnRepo repository.CdnRepositoryInterface) EdgeServiceInterface {
	return &edgeService{
		edgeRepository: edgeRepo,
		cdnRepository:  cdnRepo,
	}
}

func (s *edgeService) Register(c *gin.Context) {
	var edge domain.Edge
	if err := c.ShouldBindJSON(&edge); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	s.edgeRepository.Set(edge)
	fmt.Println(s.edgeRepository.GetAll())
	c.JSON(200, gin.H{
		"status":           "registered",
		"instance":         edge.Instance,
		"cdn_list_version": s.cdnRepository.GetVersion(),
	})
}

func (s *edgeService) GetCdns(c *gin.Context) {
	cdns := s.cdnRepository.GetAll()
	c.JSON(http.StatusOK, cdns)
}
