FROM python:3.11-slim
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
RUN pip install openviking
ENV OPENVIKING_CONFIG_FILE=/data/ov.conf
EXPOSE 1933
CMD ["python","-m","openviking","serve","--port","1933","--config","/data/ov.conf"]
