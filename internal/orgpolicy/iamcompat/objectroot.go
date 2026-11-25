package iamcompat

import "github.com/zitadel/zitadel/internal/eventstore/v1/models"

func objectRoot(instanceID, resourceOwner string) models.ObjectRoot {
	return models.ObjectRoot{
		InstanceID:    instanceID,
		ResourceOwner: resourceOwner,
	}
}
