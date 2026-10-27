FROM python:3.10-slim

WORKDIR /app

# 系统依赖：切换阿里源加速
RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources 2>/dev/null; \
    sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list 2>/dev/null; \
    apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    libgl1 \
    libglib2.0-0 \
    libxcb1 \
    libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

# 先装依赖层（利用 Docker 缓存）
COPY requirements.txt .
RUN pip install --no-cache-dir \
    -i https://mirrors.aliyun.com/pypi/simple/ \
    -r requirements.txt

# 代码层（变更频繁，放后面）
ARG APP_BUILD_VERSION=dev
RUN echo "$APP_BUILD_VERSION" > /app/APP_BUILD_VERSION
COPY scripts/ scripts/
COPY static/ static/
COPY config/ config/

# 入口
EXPOSE 8000
CMD ["uvicorn", "scripts.app:app", "--host", "0.0.0.0", "--port", "8000"]
