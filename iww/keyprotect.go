package iww

import (
	"context"
	"log"

	kp "github.com/IBM/keyprotect-go-client"
)

type ResourceFinderKeyProtect struct{}

// --- Find does does not find new resources, it does introduce a new destroy operation
func getKeyProtectClient(crn *Crn) (*kp.Client, context.Context, error) {
	gc := MustGlobalContext()
	ctx := context.Background()
	region := crn.region
	if gc.token != "" {
		config := kp.ClientConfig{
			BaseURL:    ApiEndpoint("https://<region>.kms.cloud.ibm.com", region),
			TokenURL:   kp.DefaultTokenURL,
			InstanceID: crn.id,
			Verbose:    kp.VerboseFailOnly,
		}
		client, err := kp.New(config, kp.DefaultTransport())
		ctx = kp.NewContextWithAuth(ctx, "bearer "+gc.token)
		return client, ctx, err
	} else {
		config := kp.ClientConfig{
			BaseURL:    ApiEndpoint("https://<region>.kms.cloud.ibm.com", region),
			APIKey:     gc.apikey,
			TokenURL:   kp.DefaultTokenURL,
			InstanceID: crn.id,
			Verbose:    kp.VerboseFailOnly,
		}
		client, err := kp.New(config, kp.DefaultTransport())
		return client, ctx, err
	}
}

func (finder ResourceFinderKeyProtect) Find(wrappedResourceInstances []*ResourceInstanceWrapper) (moreInstanceWrappers []*ResourceInstanceWrapper, err error) {
	MustGlobalContext().verboseLogger.Println("find ResourceFinderKeyProtect")
	moreInstanceWrappers = wrappedResourceInstances
	for _, ri := range wrappedResourceInstances {
		crn := ri.crn
		if crn.resourceType == "kms" {
			if client, ctx, err1 := getKeyProtectClient(crn); err1 == nil {
				pageSize := 3
				keys := make([]kp.Key, 0)
				// 100 times through max, avoid infinite loop
				for i := 0; i < 100; i = i + 1 {
					var getKeys *kp.Keys
					getKeys, err2 := client.GetKeys(ctx, pageSize, i*pageSize)
					if err2 == nil {
						keys = append(keys, getKeys.Keys...)
						if len(getKeys.Keys) < pageSize {
							break
						}
					} else {
						log.Println("KeyprotectServiceOpertions GetKeys failed err:", err2)
						err = err2
						return // err
					}
				}
				for _, key := range keys {
					moreInstanceWrappers = append(moreInstanceWrappers, NewSubInstance(ri, "key", key.ID, &key.Name, &KeyProtectKeyOpertions{}))
				}
				err = nil
			} else {
				err = err1
				return // err
			}
		}
	}
	return
}

// --------------------------------------
type KeyProtectKeyOpertions struct {
}

func (s *KeyProtectKeyOpertions) Fetch(si *ResourceInstanceWrapper) {
	crn := si.crn
	// todo id := s.key
	id := crn.vpcId
	if client, ctx, err := getKeyProtectClient(crn); err == nil {
		if key, err := client.GetKey(ctx, id); err != nil {
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
	if client, ctx, err := getKeyProtectClient(crn); err == nil {
		_, err := client.DeleteKey(ctx, id, kp.ReturnRepresentation, kp.ForceOpt{Force: true})
		if err != nil {
			log.Print("KeyprotectServiceOpertions error while deleting the key: ", err)
		}
	} else {
		log.Print("Error getKeyProtectClient, err:", err)
	}
}
