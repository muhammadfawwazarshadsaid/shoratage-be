# File: yolo-server/python_predictor/Dockerfile

# Gunakan base image python yang slim
FROM python:3.11-slim-bookworm

WORKDIR /app

COPY requirements.txt .

RUN apt-get update && apt-get install -y libgl1-mesa-glx libglib2.0-0 && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 5001

ENV FLASK_APP=api.py

CMD ["flask", "run", "--host=0.0.0.0", "--port=5001"]