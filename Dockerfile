FROM scratch

USER 5000
COPY gota /bin/gota
CMD [ "/bin/gota" ]