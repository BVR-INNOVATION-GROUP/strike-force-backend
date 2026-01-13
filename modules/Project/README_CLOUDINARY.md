# Cloudinary MOU Deletion

This module includes functionality to delete MOU PDFs from Cloudinary when a project is deapproved.

## Required Environment Variables

For MOU deletion to work, the following environment variables must be set in your `.env` file:

```env
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret
```

These credentials can be found in your Cloudinary Dashboard under **Settings** → **Security**.

## How It Works

When a project status is changed to `"draft"` (deapproved):

1. The system extracts the `public_id` from the MOU Cloudinary URL
2. It calls the Cloudinary Admin API to delete the file
3. It clears the `mou_url` and `university_admin_signature` fields in the database

## Error Handling

If Cloudinary deletion fails (e.g., due to missing credentials or network issues), the error is logged but the status update still proceeds. This ensures that project deapproval is not blocked by Cloudinary issues.

## Testing

To test MOU deletion:

1. Approve a project (this creates an MOU PDF)
2. Deapprove the project (status = "draft")
3. Check Cloudinary Media Library to verify the MOU PDF has been deleted
4. Check the database to verify `mou_url` and `university_admin_signature` are cleared









