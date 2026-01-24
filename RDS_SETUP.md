# RDS Production Database Setup Guide

This guide explains how to configure your application to connect to Amazon RDS PostgreSQL in production.

## Required Changes

### 1. GitHub Secrets Configuration

Add/update the following secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):

#### Database Connection Secrets:
- `DB_HOST` - Your RDS endpoint (e.g., `strikeforce-db.xxxxx.us-east-1.rds.amazonaws.com`)
- `DB_PORT` - PostgreSQL port (usually `5432`)
- `DB_USER` - RDS master username
- `DB_PASSWORD` - RDS master password
- `DB_NAME` - Database name (e.g., `strikeforce_db`)
- `DB_SSLMODE` - **Set to `require`** (required for RDS connections)

#### Example Values:
```
DB_HOST=strikeforce-db.xxxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=YourSecurePassword123!
DB_NAME=strikeforce_db
DB_SSLMODE=require
```

### 2. RDS Security Group Configuration

Ensure your RDS security group allows inbound connections from your EC2 instance:

1. Go to AWS Console → RDS → Your Database → Connectivity & Security
2. Click on the Security Group
3. Edit Inbound Rules
4. Add rule:
   - **Type**: PostgreSQL
   - **Port**: 5432
   - **Source**: Your EC2 security group ID (or EC2 private IP)

**Recommended**: Use the EC2 security group as the source for better security.

### 3. RDS Database Configuration

When creating/updating your RDS instance:

- **Engine**: PostgreSQL (version 15 or compatible)
- **Publicly accessible**: Set to **No** (for security - only accessible from EC2)
- **VPC**: Same VPC as your EC2 instance
- **Subnet group**: Use default or create one with private subnets
- **Database name**: Match your `DB_NAME` secret
- **Master username**: Match your `DB_USER` secret
- **Master password**: Match your `DB_PASSWORD` secret

### 4. SSL Mode Options

The `DB_SSLMODE` environment variable supports these values:

- `disable` - No SSL (local development only)
- `require` - **Recommended for RDS** - Requires SSL but doesn't verify certificate
- `verify-ca` - Requires SSL and verifies CA certificate
- `verify-full` - Requires SSL and verifies CA and hostname (most secure)

For RDS, use `require` unless you have SSL certificates configured.

### 5. Manual Docker Run (if not using CI/CD)

If you need to manually run the container on EC2:

```bash
docker run -d \
  --name strikeforcev1 \
  -p 3003:3000 \
  -e SECRET_KEY=your-secret-key \
  -e APP_PORT=3000 \
  -e NEXT_PUBLIC_API_URL=https://your-domain.com \
  -e DB_NAME=strikeforce_db \
  -e DB_USER=admin \
  -e DB_PASSWORD=YourSecurePassword123! \
  -e DB_PORT=5432 \
  -e DB_HOST=strikeforce-db.xxxxx.us-east-1.rds.amazonaws.com \
  -e DB_SSLMODE=require \
  -e MAILJET_KEY=your-key \
  -e MAILJET_SECRET=your-secret \
  -e MAILJET_EMAIL=your-email \
  -e MAILJET_FROM=strikeforce \
  -e SUPER_ADMIN_EMAIL=admin@strikeforce.com \
  -e SUPER_ADMIN_PASSWORD=admin123 \
  --restart always \
  ssenkootokigongovincent/strikeforcev1:latest
```

## Troubleshooting

### Connection Issues

1. **"hostname resolving error"**
   - Check that `DB_HOST` is set correctly (full RDS endpoint)
   - Verify RDS instance is running

2. **"password authentication failed"**
   - Verify `DB_USER` and `DB_PASSWORD` match RDS credentials
   - Check RDS master username/password

3. **"connection timeout"**
   - Verify security group allows connections from EC2
   - Check that RDS and EC2 are in the same VPC
   - Ensure RDS is not publicly accessible (should be private)

4. **"SSL connection required"**
   - Set `DB_SSLMODE=require` in your environment variables
   - RDS requires SSL connections

### Testing Connection from EC2

SSH into your EC2 instance and test the connection:

```bash
# Install PostgreSQL client (if not already installed)
sudo yum install postgresql15 -y

# Test connection
psql -h your-rds-endpoint.xxxxx.us-east-1.rds.amazonaws.com \
     -U admin \
     -d strikeforce_db \
     -p 5432
```

## Verification

After deployment, check the application logs:

```bash
docker logs strikeforcev1
```

You should see:
```
Connecting to database...
Connected to DB successfully
Database connected successfully
```

If you see errors, check:
1. All environment variables are set correctly
2. Security group rules allow connection
3. RDS instance is running and accessible
4. SSL mode is set to `require`
