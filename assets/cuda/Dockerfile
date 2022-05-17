FROM python:3.8.13-slim-buster
COPY . /src
RUN pip install -r /src/requirements.txt
CMD [ "python", "/src/server.py" ]