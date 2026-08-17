FROM mcr.microsoft.com/playwright:v1.46.0-jammy
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev
COPY src ./src
EXPOSE 3000
CMD ["npm", "start"]

