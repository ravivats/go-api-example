# Docker and Docker Compose Installation Guide

## Install Docker

### 1. Update package index
- **Ubuntu-based systems:**
  ```bash
  sudo apt update
  ```
- **RHEL-based systems:**
  ```bash
  sudo yum update
  ```

### 2. Install Docker
- **Ubuntu-based systems:**
  ```bash
  sudo apt install docker.io
  ```
- **RHEL-based systems:**
  ```bash
  sudo yum install docker
  ```
- **macOS (with Homebrew):**
  ```bash
  brew install --cask docker
  ```
- **Windows:**
  - Download and install Docker Desktop from the official Docker website:  
    [https://www.docker.com/products/docker-desktop/](https://www.docker.com/products/docker-desktop/)

### 3. Start Docker service
- **Ubuntu-based systems:**
  ```bash
  sudo systemctl start docker
  ```
- **RHEL-based systems:**
  ```bash
  sudo systemctl start docker
  ```

### 4. Verify Docker installation
```bash
docker --version
```

---

## Install Docker Compose

### Method 1: Download Binary (Recommended)

1. **Download Docker Compose binary**
   ```bash
   sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   ```

2. **Apply executable permissions**
   ```bash
   sudo chmod +x /usr/local/bin/docker-compose
   ```

3. **Verify Docker Compose installation**
   ```bash
   docker-compose --version
   ```

### Method 2: Using pip (Alternative)

1. **Install pip**
   - **Ubuntu-based systems:**
     ```bash
     sudo apt install python3-pip
     ```
   - **RHEL-based systems:**
     ```bash
     sudo yum install python3-pip
     ```

2. **Install Docker Compose via pip**
   ```bash
   pip3 install docker-compose
   ```

3. **Verify Docker Compose installation**
   ```bash
   docker-compose --version
   ```

---

## Post-installation Steps

- Add your user to the Docker group (optional):
  ```bash
  sudo usermod -aG docker $USER
  ```

- Log out and log back in (or restart your session) for group changes to take effect.