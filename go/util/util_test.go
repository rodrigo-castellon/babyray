package util

import (
    "context"
    "testing"

    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func TestExtractAddressFromCtx(t *testing.T) {
    // Create a context with metadata
    md := metadata.New(map[string]string{"client-address": "192.168.1.1"})
    ctx := metadata.NewIncomingContext(context.Background(), md)

    // Test extraction
    addr, err := extractAddressFromCtx(ctx)
    if err != nil {
        t.Fatalf("Failed to extract address: %v", err)
    }
    if addr != "192.168.1.1" {
        t.Errorf("Extracted address is incorrect, got: %s, want: %s", addr, "192.168.1.1")
    }

    // Test with no metadata
    ctxNoMeta := context.Background()
    _, err = extractAddressFromCtx(ctxNoMeta)
    if err == nil {
        t.Errorf("Expected an error for missing metadata, but got none")
    }
    st, _ := status.FromError(err)
    if st.Code() != codes.InvalidArgument {
        t.Errorf("Expected InvalidArgument, got: %v", st.Code())
    }
}
