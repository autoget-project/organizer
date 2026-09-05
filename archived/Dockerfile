# Use Python 3.13 slim image
FROM python:3.13-slim

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_CACHE_DIR=/tmp/uv-cache

# Install system dependencies and uv
RUN apt-get update && apt-get install -y \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && pip install uv

# Create non-root user
RUN useradd -u 99 --create-home --shell /bin/bash app

# Create /mnt directory and assign to user 99, both download and target should
# be mounted under this.
RUN mkdir -p /mnt && chown 99:100 /mnt

USER app

# Set work directory
WORKDIR /app

# Copy all files
COPY . .

# Install dependencies
RUN uv sync --frozen --no-dev

# Expose port
EXPOSE 8000

# Run the application
CMD ["uv", "run", "fastapi", "dev", "--host", "0.0.0.0", "--port", "8000"]
