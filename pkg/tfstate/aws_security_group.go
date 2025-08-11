package tfstate

const (
	TypeAwsSecurityGroup     = "aws_security_group"
	TypeAwsSecurityGroupRule = "aws_security_group_rule"
)

type AwsSecurityGroup struct {
	Arn     string            `json:"arn"`
	Egress  []EgressItem      `json:"egress"`
	Ingress []IngressItem     `json:"ingress"`
	Id      string            `json:"id"`
	Name    string            `json:"name"`
	OwnerId string            `json:"owner_id"`
	TagsAll map[string]string `json:"tags_all"`
	VpcId   string            `json:"vpc_id"`
}

type EgressItem map[string]any

type IngressItem struct {
	CidrBlocks     []string `json:"cidr_blocks"`
	Description    string   `json:"description"`
	FromPort       int      `json:"from_port"`
	Ipv6CidrBlocks []any    `json:"ipv6_cidr_blocks"`
	PrefixListIds  []any    `json:"prefix_list_ids"`
	Protocol       string   `json:"protocol"`
	SecurityGroups []any    `json:"security_groups"`
	Self           bool     `json:"self"`
	ToPort         int      `json:"to_port"`
}

type AwsSecurityGroupRule struct {
}
