FROM --platform=linux/amd64 node:18
USER root
WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm install

# Copy source code
COPY . .

# Build TypeScript
RUN npm run build

# Expose API port
EXPOSE 3000

# Start the validation service
CMD ["npm", "start"]
