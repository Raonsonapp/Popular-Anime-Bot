FROM python:3.11-slim
WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# The Telethon session file is created by running login.py interactively
# once (outside Docker) and then mounted here read-write so the container
# can reconnect without a TTY.
VOLUME ["/data"]

CMD ["python", "main.py"]
