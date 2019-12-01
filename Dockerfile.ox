FROM scratch

COPY bin/ox .

ENTRYPOINT ["./ox"]

EXPOSE 2223