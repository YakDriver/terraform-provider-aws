package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

// Route table import also imports all the rules
func resourceAwsRouteTableImportState(
	d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*AWSClient).ec2conn

	// First query the resource itself
	id := d.Id()
	resp, err := conn.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&id},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.RouteTables) < 1 || resp.RouteTables[0] == nil {
		return nil, fmt.Errorf("route table %s is not found", id)
	}

	table := resp.RouteTables[0]
	if len(table.Routes) > 0 {
		log.Print("[WARN] Routes in the route table so assuming they are defined inline.")
		err := resourceAwsRouteTableRead(d, meta)
		if err != nil {
			return nil, err
		}

		results := make([]*schema.ResourceData, 1)
		results[0] = d
		return results, nil
	}

	log.Print("[WARN] No routes in the route table so assuming they are defined separately.")

	// Start building our results
	results := make([]*schema.ResourceData, 2)
	results[0] = d
	log.Print("[WARN] RouteTable imports will be handled differently in a future version.")
	log.Printf("[WARN] This import will create %d resources (aws_route_table, aws_route, aws_route_table_association).", len(results))
	log.Print("[WARN] In the future, only 1 aws_route_table resource will be created with inline routes.")

	{
		// Construct the main associations. We could do this above but
		// I keep this as a separate section since it is a separate resource.
		subResource := resourceAwsMainRouteTableAssociation()
		for _, assoc := range table.Associations {
			if !*assoc.Main {
				// Ignore
				continue
			}

			// Minimal data for route
			d := subResource.Data(nil)
			d.SetType("aws_main_route_table_association")
			d.Set("route_table_id", id)
			d.Set("vpc_id", table.VpcId)
			d.SetId(*assoc.RouteTableAssociationId)
			results = append(results, d)
		}
	}

	return results, nil
}
