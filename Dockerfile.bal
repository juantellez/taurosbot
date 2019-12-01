FROM python:3

WORKDIR /usr/src/app

COPY bal/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY bal/*.py ./

CMD ["python", "./balances.py"]

EXPOSE 2224
