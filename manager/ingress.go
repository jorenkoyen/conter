package manager

import (
	"fmt"
	"github.com/jorenkoyen/conter/manager/db"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"slices"
)

type IngressManager struct {
	logger             *logger.Logger
	Database           *db.Client
	CertificateManager *CertificateManager
}

func NewIngressManager() *IngressManager {
	return &IngressManager{
		logger: log.WithName("ingress-mgr"),
	}
}

// RegisterRoute will register a new ingress route and complete the necessary actions to make it ready for use.
func (i *IngressManager) RegisterRoute(ingress types.Ingress) error {
	i.logger.Debugf("Registering route for domain=%s (endpoint=%s, challenge_type=%s)", ingress.Domain, ingress.TargetEndpoint, ingress.ChallengeType)

	// check if we already have a registered route
	//	-> update endpoint (if required)
	// 	-> check if project is correct
	route, err := i.Match(ingress.Domain)
	if err == nil {
		if route.TargetEndpoint == ingress.TargetEndpoint {
			// no action required, correct endpoint already assigned
			i.logger.Debugf("No action required for registering route for %s, endpoint is already correct", ingress.Domain)
			return nil
		}
		if route.TargetProject != ingress.TargetProject {
			return fmt.Errorf("domain %s is already in use for project=%s", ingress.Domain, route.TargetProject)
		}
		if route.TargetService != ingress.TargetService {
			i.logger.Warningf("Overwriting domain configuration for service=%s, now pointing to service=%s (domain=%s)", route.TargetService, ingress.TargetService, ingress.Domain)
		}
	}

	// save ingress route before requesting challenge (avoid race conditions)
	err = i.Database.SaveIngressRoute(&ingress)
	if err != nil {
		return fmt.Errorf("failed to save ingress route: %w", err)
	}

	// kick-off challenge creation for ingress
	i.CertificateManager.ChallengeCreate(ingress)
	return nil
}

// RemoveUnusedRoutes will remove all unused routes related to the specified project.
func (i *IngressManager) RemoveUnusedRoutes(project string, excludedDomains []string) (int, error) {
	i.logger.Tracef("Removing unused routes for project=%s (excluded=%s)", project, excludedDomains)

	routes := i.Database.GetIngressRoutesByProject(project)
	removed := 0
	for _, route := range routes {
		if slices.Index(excludedDomains, route.Domain) == -1 {
			// not listed in excluded domains -> remove
			i.logger.Debugf("Removing unused route for domain=%s linked to project=%s", route.Domain, project)
			err := i.Database.RemoveIngressRoute(route.Domain)
			if err != nil {
				return removed, fmt.Errorf("failed to remove %s: %w", route.Domain, err)
			}

			removed++
		}
	}

	return removed, nil
}

// RemoveAllRoutes will remove all routes linked to the specified project.
func (i *IngressManager) RemoveAllRoutes(project string) (int, error) {
	return i.RemoveUnusedRoutes(project, []string{}) // no excluded domains
}

// Match will retrieve the ingress route information for the specified domain.
func (i *IngressManager) Match(domain string) (*types.Ingress, error) {
	return i.Database.GetIngressRoute(domain)
}
