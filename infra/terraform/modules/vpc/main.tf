resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = { Name = "edugraph-${var.environment}" }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "edugraph-${var.environment}-igw" }
}

# ── Public subnets (ALB lives here) ─────────────────────────────────────────
resource "aws_subnet" "public" {
  count                   = length(var.availability_zones)
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidrs[count.index]
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name                                          = "edugraph-${var.environment}-public-${count.index + 1}"
    "kubernetes.io/role/elb"                      = "1"
    "kubernetes.io/cluster/edugraph-${var.environment}" = "shared"
  }
}

# ── Private subnets (EKS nodes, ElastiCache, Neo4j EC2) ─────────────────────
resource "aws_subnet" "private" {
  count             = length(var.availability_zones)
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]

  tags = {
    Name                                          = "edugraph-${var.environment}-private-${count.index + 1}"
    "kubernetes.io/role/internal-elb"             = "1"
    "kubernetes.io/cluster/edugraph-${var.environment}" = "shared"
  }
}

# ── NAT Gateway (single, in first public subnet) ─────────────────────────────
resource "aws_eip" "nat" {
  domain = "vpc"
  tags   = { Name = "edugraph-${var.environment}-nat-eip" }
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public[0].id
  tags          = { Name = "edugraph-${var.environment}-nat" }
  depends_on    = [aws_internet_gateway.main]
}

# ── Route tables ──────────────────────────────────────────────────────────────
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }
  tags = { Name = "edugraph-${var.environment}-public-rt" }
}

resource "aws_route_table_association" "public" {
  count          = length(aws_subnet.public)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }
  tags = { Name = "edugraph-${var.environment}-private-rt" }
}

resource "aws_route_table_association" "private" {
  count          = length(aws_subnet.private)
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private.id
}

# ── Security Groups ───────────────────────────────────────────────────────────
resource "aws_security_group" "alb" {
  name        = "edugraph-${var.environment}-alb"
  description = "Allow inbound HTTP/HTTPS from internet"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "edugraph-${var.environment}-alb-sg" }
}

resource "aws_security_group" "eks_nodes" {
  name        = "edugraph-${var.environment}-eks-nodes"
  description = "Allow traffic between EKS nodes and from ALB"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "Allow from ALB"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }
  ingress {
    description     = "Allow from ALB to frontend"
    from_port       = 80
    to_port         = 80
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }
  ingress {
    description = "Node-to-node communication"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "edugraph-${var.environment}-eks-nodes-sg" }
}

resource "aws_security_group" "redis" {
  name        = "edugraph-${var.environment}-redis"
  description = "Allow Redis access from EKS nodes only"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_nodes.id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "edugraph-${var.environment}-redis-sg" }
}

resource "aws_security_group" "neo4j" {
  name        = "edugraph-${var.environment}-neo4j"
  description = "Allow Neo4j bolt access from EKS nodes only"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "Bolt protocol from EKS"
    from_port       = 7687
    to_port         = 7687
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_nodes.id]
  }
  ingress {
    description     = "HTTP browser (internal only)"
    from_port       = 7474
    to_port         = 7474
    protocol        = "tcp"
    security_groups = [aws_security_group.eks_nodes.id]
  }
  # SSH for maintenance — restrict to your VPN/bastion CIDR in practice
  ingress {
    description = "SSH (restrict to VPN CIDR in production)"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "edugraph-${var.environment}-neo4j-sg" }
}
