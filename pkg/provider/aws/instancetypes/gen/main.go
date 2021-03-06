// +build ignore

package main

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
)

type Instance struct {
	Product ProductAttributes `json:"product"`
	Terms   Terms             `json:"terms"`
}

type Terms struct {
	OnDemand map[string]PriceTypes `json:"OnDemand"`
}

type PriceTypes struct {
	EffectiveDate   string           `json:"effectiveDate"`
	OfferTermCode   string           `json:"offerTermCode"`
	SKU             string           `json:"sku"`
	PriceDimensions map[string]Price `json:"priceDimensions"`
}

type Price struct {
	Description  string            `json:"description"`
	PricePerUnit map[string]string `json:"pricePerUnit"`
	Unit         string            `json:"unit"`
}

type ProductAttributes struct {
	Attributes EC2Attributes `json:"attributes"`
}

type EC2Attributes struct {
	ServiceCode       string `json:"servicecode"`
	ServiceName       string `json:"servicename"`
	InstanceFamily    string `json:"instanceFamily"`
	InstanceType      string `json:"instanceType"`
	Location          string `json:"location"`
	VCPU              string `json:"vcpu"`
	GPU               string `json:"gpu"`
	Memory            string `json:"memory"`
	PhysicalProcessor string `json:"physicalProcessor"`
	Storage           string `json:"storage"`
}

type VM struct {
	Name        string
	VCPU        string
	MemoryGiB   string
	GPU         string
	PriceHour   string
	Description string
}

var packageTemplate = template.Must(template.New("").Parse(`// This file was generated by go generate; DO NOT EDIT
package instancetypes

// RegionTypes returns a list of supported vm types in the region.
func RegionTypes(region string) ([]VM, error) {
	return awsMachines.regionTypes(region)
}

var awsMachines = manager{
	regionVMs: map[string][]VM{
{{- range $k, $v := .RegionInstances }}
		"{{ $k }}": {
{{- range $v }}
			{
				Name:        "{{ .InstanceType }}",
				VCPU:        "{{ .VCPU }}",
				MemoryGiB:   "{{ .MemoryGiB }}",
				GPU:         "{{ .GPU }}",
				PriceHour:   "{{ .PriceHour }}",
				Description: "{{ .Description }}",
			},
{{- end }}
		},
{{- end }}
	},
}
`))

type instanceType struct {
	InstanceType string
	VCPU         string
	MemoryGiB    string
	GPU          string
	PriceHour    string
	Description  string
}

func main() {
	resolver := endpoints.DefaultResolver()

	regionTypes := make(map[string][]instanceType)
	for _, p := range resolver.(endpoints.EnumPartitions).Partitions() {
		for _, r := range p.Regions() {
			log.Println("Retrieve instance attributes for:", r.ID())
			vms, err := getEC2types(r.Description())
			handle(err)

			for _, vm := range vms {
				if regionTypes[r.ID()] == nil {
					regionTypes[r.ID()] = make([]instanceType, 0, len(vms))
				}
				t := instanceType{
					InstanceType: vm.Name,
					VCPU:         vm.VCPU,
					MemoryGiB:    parseMemory(vm.MemoryGiB),
					GPU:          vm.GPU,
					PriceHour:    vm.PriceHour,
					Description:  vm.Description,
				}
				regionTypes[r.ID()] = append(regionTypes[r.ID()], t)
			}
		}
	}

	for region := range regionTypes {
		sort.Slice(regionTypes[region], func(i, j int) bool {
			return regionTypes[region][i].InstanceType < regionTypes[region][j].InstanceType
		})
	}

	f, err := os.Create("vm_types_aws.go")
	if err != nil {
		handle(err)
	}

	defer f.Close()

	err = packageTemplate.Execute(f, struct {
		RegionInstances map[string][]instanceType
	}{
		RegionInstances: regionTypes,
	})

	if err != nil {
		handle(err)
	}
}

func getEC2types(location string) ([]VM, error) {
	svc := pricing.New(session.New(), &aws.Config{Region: aws.String("us-east-1")})
	input := &pricing.GetProductsInput{
		Filters: []*pricing.Filter{
			{
				Field: aws.String("ServiceCode"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("AmazonEC2"),
			},
			{
				Field: aws.String("location"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String(location),
			},
			{
				Field: aws.String("productFamily"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Compute Instance"),
			},
			{
				Field: aws.String("termType"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("OnDemand"),
			},
			{
				Field: aws.String("operatingSystem"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Linux"),
			},
			{
				Field: aws.String("operation"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("RunInstances"),
			},
			{
				Field: aws.String("tenancy"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("Shared"),
			},
			{
				Field: aws.String("capacitystatus"),
				Type:  aws.String("TERM_MATCH"),
				Value: aws.String("UnusedCapacityReservation"), // https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-capacity-reservations.html
				                                                  // https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/capacity-reservations-pricing-biling.html
			},
		},
		FormatVersion: aws.String("aws_v1"),
		MaxResults:    aws.Int64(5),
		ServiceCode:   aws.String("AmazonEC2"),
	}

	priceList := make([]aws.JSONValue, 0)
	for {
		result, err := svc.GetProducts(input)
		if err != nil {
			return nil, err
		}
		priceList = append(priceList, result.PriceList...)
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}

	ec2types := make([]VM, 0)
	for _, product := range priceList {
		raw, _ := json.Marshal(product)

		res := Instance{}
		err := json.Unmarshal(raw, &res)
		if err != nil {
			return nil, err
		}

		for _, priceType := range res.Terms.OnDemand {
			for _, price := range priceType.PriceDimensions {
				for currency, amount := range price.PricePerUnit {
					if currency == "USD" {
						ec2types = append(ec2types, VM{
							Name:        res.Product.Attributes.InstanceType,
							VCPU:        res.Product.Attributes.VCPU,
							MemoryGiB:   res.Product.Attributes.Memory,
							GPU:         res.Product.Attributes.GPU,
							PriceHour:   amount,
							Description: price.Description,
						})
					}
				}
			}
		}
	}

	return ec2types, nil
}

func toStrings(in []*pricing.AttributeValue) []string {
	o := make([]string, len(in))
	for i, val := range in {
		if val != nil {
			o[i] = aws.StringValue(val.Value)
		}
	}
	return o
}

func parseMemory(memory string) string {
	reg, err := regexp.Compile("[^0-9\\.]+")
	handle(err)

	return strings.TrimSpace(reg.ReplaceAllString(memory, ""))
}

func handle(err error) {
	if err == nil {
		return
	}
	log.Fatal(err)
}
