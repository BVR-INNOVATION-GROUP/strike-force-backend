# Production Deployment Setup

This deployment configuration uses environment variables from a `.env.production` file on your EC2 instance, eliminating the need for GitHub secrets.

## Setup Instructions

### 1. Create `.env.production` file on EC2

SSH into your EC2 instance and create the environment file:

```bash
ssh ec2-user@your-ec2-ip
cd ~
nano .env.production
```

### 2. Add your production environment variables

Copy the template from `.env.production.example` and fill in your actual values:

```bash
SECRET_KEY=your-secret-key-here
APP_PORT=3000
NEXT_PUBLIC_API_URL=https://your-domain.com

# RDS Database Configuration
DB_HOST=your-rds-endpoint.xxxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=your-secure-password-here
DB_NAME=strikeforce_db
DB_SSLMODE=require

# Mailjet Configuration
MAILJET_KEY=your-mailjet-key
MAILJET_SECRET=your-mailjet-secret
MAILJET_EMAIL=your-email@example.com
MAILJET_FROM=strikeforce

# Super Admin Configuration
SUPER_ADMIN_EMAIL=admin@strikeforce.com
SUPER_ADMIN_PASSWORD=your-admin-password
```

### 3. Set proper file permissions

```bash
chmod 600 /home/ec2-user/.env.production
```

This ensures only the owner can read/write the file (important for security).

### 4. Verify the file exists

```bash
ls -la /home/ec2-user/.env.production
```

You should see:
```
-rw------- 1 ec2-user ec2-user 450 Jan 24 20:00 /home/ec2-user/.env.production
```

## Deployment

Once the `.env.production` file is set up on your EC2 instance, the GitHub Actions workflow will automatically:

1. Pull the latest Docker image
2. Stop and remove the old container
3. Start a new container using environment variables from `/home/ec2-user/.env.production`

## Updating Environment Variables

To update environment variables:

1. SSH into EC2
2. Edit the `.env.production` file:
   ```bash
   nano /home/ec2-user/.env.production
   ```
3. Restart the container:
   ```bash
   docker restart strikeforcev1
   ```

Or let the next deployment handle it automatically.

## Required GitHub Secrets

You still need these secrets for Docker Hub and EC2 access:

- `DOCKER_USERNAME` - Your Docker Hub username
- `DOCKER_PAT` - Docker Hub personal access token
- `EC2_HOST` - EC2 instance IP or hostname
- `EC2_SECRET` - SSH private key for EC2 access

## Troubleshooting

### Container fails to start

Check the logs:
```bash
docker logs strikeforcev1
```

Common issues:
- Missing `.env.production` file → Create it following step 1-2 above
- Invalid environment variables → Check the file format (no spaces around `=`)
- Database connection issues → Verify RDS endpoint and credentials

### Verify environment variables are loaded

```bash
docker exec strikeforcev1 env | grep DB_
```

This will show all database-related environment variables loaded in the container.
