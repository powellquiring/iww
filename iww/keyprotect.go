package iww

import (
	"context"
	"log"

	kp "github.com/IBM/keyprotect-go-client"
)

// --- Find does does not find new resources, it does introduce a new destroy operation
type ResourceFinderKeyProtect struct{}

func (finder ResourceFinderKeyProtect) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	MustGlobalContext().verboseLogger.Println("find ResourceFinderKeyProtect")
	moreInstanceWrappers = wrappedResourceInstances
	var client *kp.Client
	for _, ri := range wrappedResourceInstances {
		crn := ri.crn
		if crn.resourceType == "kms" {
			if client, err = MustGlobalContext().getKeyProtectClient(crn); err == nil {
				pageSize := 3
				keys := make([]kp.Key, 0)
				// 100 times through max, avoid infinite loop
				for i := 0; i < 100; i = i + 1 {
					var getKeys *kp.Keys
					getKeys, err = client.GetKeys(context.Background(), pageSize, i*pageSize)
					if err == nil {
						keys = append(keys, getKeys.Keys...)
						if len(getKeys.Keys) < pageSize {
							break
						}
					} else {
						log.Println("KeyprotectServiceOpertions GetKeys failed err:", err)
						return // err
					}
				}
				for _, key := range keys {
					moreInstanceWrappers = append(moreInstanceWrappers, NewSubInstance(ri, "key", key.ID, &key.Name, &KeyProtectKeyOpertions{}))
				}
			} else {
				return // err
			}
		}
	}
	return
}

//--------------------------------------
type KeyProtectKeyOpertions struct {
}

func (s *KeyProtectKeyOpertions) Fetch(si *ResourceInstanceWrapper) {
	crn := si.crn
	// todo id := s.key
	id := crn.vpcId
	if client, err := MustGlobalContext().getKeyProtectClient(crn); err == nil {
		if key, err := client.GetKey(context.Background(), id); err != nil {
			si.state = SIStateDeleted
		} else {
			si.Name = &key.Name
			si.state = SIStateExists
		}
	}
}

func (s *KeyProtectKeyOpertions) FormatInstance(si *ResourceInstanceWrapper, fast bool) string {
	return FormatInstance(*si.Name, "kp key", *si.crn)
}

func (s *KeyProtectKeyOpertions) Destroy(si *ResourceInstanceWrapper) {
	crn := si.crn
	id := crn.vpcId
	if client, err := MustGlobalContext().getKeyProtectClient(crn); err == nil {
		_, err := client.DeleteKey(context.Background(), id, kp.ReturnRepresentation, kp.ForceOpt{Force: true})
		if err != nil {
			log.Print("KeyprotectServiceOpertions error while deleting the key: ", err)
		}
	} else {
		log.Print("Error getKeyProtectClient, err:", err)
	}
}
