FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-percipio"]
COPY baton-percipio /