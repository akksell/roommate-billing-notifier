FROM gcr.io/distroless/base-debian11 as release-stage
ENV ENVIRONMENT=production

WORKDIR /

COPY ./build/rbn rbn

USER nonroot:nonroot

ENTRYPOINT ["./rbn"]
