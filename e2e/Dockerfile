# Gunakan official image Playwright yang sudah memuat semua browser binaries & OS dependencies
FROM mcr.microsoft.com/playwright:v1.65.1-noble

WORKDIR /app

# Salin package.json & tsconfig
COPY package.json package-lock.json tsconfig.json playwright.config.ts ./

# Install dependencies (Playwright package)
RUN npm ci

# Salin fixtures & tests
COPY fixtures/ ./fixtures/
COPY tests/ ./tests/

# Default environment
ENV CI=true
ENV BASE_URL=http://host.docker.internal:3000

# Eksekusi test dan hasilkan laporan ke /app/playwright-report
CMD ["npx", "playwright", "test"]
