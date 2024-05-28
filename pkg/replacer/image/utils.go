//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package image

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/stacklok/frizbee/internal/cli"
	"github.com/stacklok/frizbee/pkg/interfaces"
	"github.com/stacklok/frizbee/pkg/utils/store"
)

// GetImageDigestFromRef returns the digest of a container image reference
// from a name.Reference.
func GetImageDigestFromRef(ctx context.Context, imageRef, platform string, cache store.RefCacher) (*interfaces.EntityRef, error) {
	// Parse the image reference
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, err
	}
	opts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithUserAgent(cli.UserAgent),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	}

	// Set the platform if provided
	if platform != "" {
		platformSplit := strings.Split(platform, "/")
		if len(platformSplit) != 2 {
			return nil, errors.New("platform must be in the format os/arch")
		}
		opts = append(opts, remote.WithPlatform(v1.Platform{
			OS:           platformSplit[0],
			Architecture: platformSplit[1],
		}))
	}

	// Get the digest of the image reference
	var digest string

	if cache != nil {
		if d, ok := cache.Load(imageRef); ok {
			digest = d
		} else {
			desc, err := remote.Get(ref, opts...)
			if err != nil {
				return nil, err
			}
			digest = desc.Digest.String()
			cache.Store(imageRef, digest)
		}
	} else {
		desc, err := remote.Get(ref, opts...)
		if err != nil {
			return nil, err
		}
		digest = desc.Digest.String()
	}

	// Compare the digest with the reference and return the original reference if they already match
	if digest == ref.Identifier() {
		return nil, fmt.Errorf("image already referenced by digest: %s %w", imageRef, interfaces.ErrReferenceSkipped)
	}

	return &interfaces.EntityRef{
		Name: ref.Context().Name(),
		Ref:  digest,
		Type: ReferenceType,
		Tag:  ref.Identifier(),
	}, nil
}

func shouldExclude(ref string) bool {
	return ref == "scratch"
}
