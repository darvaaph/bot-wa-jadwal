package main

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// mockLIDStore mengimplementasikan store.LIDStore sederhana untuk unit test
type mockLIDStore struct {
	lidToPN map[string]types.JID
	pnToLID map[string]types.JID
}

func newMockLIDStore() *mockLIDStore {
	return &mockLIDStore{
		lidToPN: make(map[string]types.JID),
		pnToLID: make(map[string]types.JID),
	}
}

func (m *mockLIDStore) PutManyLIDMappings(ctx context.Context, mappings []store.LIDMapping) error {
	for _, mapping := range mappings {
		m.lidToPN[mapping.LID.ToNonAD().String()] = mapping.PN.ToNonAD()
		m.pnToLID[mapping.PN.ToNonAD().String()] = mapping.LID.ToNonAD()
	}
	return nil
}

func (m *mockLIDStore) PutLIDMapping(ctx context.Context, lid, jid types.JID) error {
	m.lidToPN[lid.ToNonAD().String()] = jid.ToNonAD()
	m.pnToLID[jid.ToNonAD().String()] = lid.ToNonAD()
	return nil
}

func (m *mockLIDStore) GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error) {
	if pn, ok := m.lidToPN[lid.ToNonAD().String()]; ok {
		return pn, nil
	}
	return types.EmptyJID, nil
}

func (m *mockLIDStore) GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error) {
	if lid, ok := m.pnToLID[pn.ToNonAD().String()]; ok {
		return lid, nil
	}
	return types.EmptyJID, nil
}

func (m *mockLIDStore) GetManyLIDsForPNs(ctx context.Context, pns []types.JID) (map[types.JID]types.JID, error) {
	res := make(map[types.JID]types.JID)
	for _, pn := range pns {
		if lid, ok := m.pnToLID[pn.ToNonAD().String()]; ok {
			res[pn] = lid
		}
	}
	return res, nil
}

func TestCheckAdminStatus(t *testing.T) {
	ctx := context.Background()

	adminPN := types.NewJID("628123456789", types.DefaultUserServer)
	adminLID := types.NewJID("264608908623896", "lid")

	admin2PN := types.NewJID("628987654321", types.DefaultUserServer)
	admin2LID := types.NewJID("987654321012345", "lid")

	ownerPN := types.NewJID("628111111111", types.DefaultUserServer)
	ownerLID := types.NewJID("111111111111111", "lid")

	normalMember := types.NewJID("628555555555", types.DefaultUserServer)
	normalMemberLID := types.NewJID("555555555555555", "lid")

	mockStore := newMockLIDStore()
	_ = mockStore.PutLIDMapping(ctx, admin2LID, admin2PN)

	groupInfo := &types.GroupInfo{
		JID:      types.NewJID("120363000000000000", types.GroupServer),
		OwnerJID: ownerLID,
		OwnerPN:  ownerPN,
		Participants: []types.GroupParticipant{
			{
				JID:         adminPN,
				PhoneNumber: adminPN,
				LID:         adminLID,
				IsAdmin:     true,
			},
			{
				JID:          admin2PN,
				PhoneNumber:  admin2PN,
				IsSuperAdmin: true,
			},
			{
				JID:          normalMember,
				PhoneNumber:  normalMember,
				LID:          normalMemberLID,
				IsAdmin:      false,
				IsSuperAdmin: false,
			},
		},
	}

	tests := []struct {
		name      string
		info      *types.GroupInfo
		sender    types.JID
		senderAlt types.JID
		store     store.LIDStore
		wantAdmin bool
	}{
		{
			name:      "Admin via Phone Number JID langsung",
			info:      groupInfo,
			sender:    adminPN,
			wantAdmin: true,
		},
		{
			name:      "Admin via LID cocok dengan p.LID",
			info:      groupInfo,
			sender:    adminLID,
			wantAdmin: true,
		},
		{
			name:      "Admin via LID dengan AD-JID device (264608908623896:0@lid)",
			info:      groupInfo,
			sender:    types.NewADJID("264608908623896", 0, 0),
			wantAdmin: true,
		},
		{
			name:      "Admin via SenderAlt Phone Number",
			info:      groupInfo,
			sender:    types.NewJID("unregistered_lid", "lid"),
			senderAlt: adminPN,
			wantAdmin: true,
		},
		{
			name:      "Admin2 via LID store resolution (p.LID kosong di group info)",
			info:      groupInfo,
			sender:    admin2LID,
			store:     mockStore,
			wantAdmin: true,
		},
		{
			name:      "Owner via OwnerPN",
			info:      groupInfo,
			sender:    ownerPN,
			wantAdmin: true,
		},
		{
			name:      "Owner via OwnerJID (LID)",
			info:      groupInfo,
			sender:    ownerLID,
			wantAdmin: true,
		},
		{
			name:      "Anggota biasa (bukan admin) via Phone Number",
			info:      groupInfo,
			sender:    normalMember,
			wantAdmin: false,
		},
		{
			name:      "Anggota biasa (bukan admin) via LID",
			info:      groupInfo,
			sender:    normalMemberLID,
			wantAdmin: false,
		},
		{
			name:      "GroupInfo nil",
			info:      nil,
			sender:    adminPN,
			wantAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckAdminStatus(ctx, tt.info, tt.sender, tt.senderAlt, tt.store)
			if got != tt.wantAdmin {
				t.Errorf("CheckAdminStatus() = %v, want %v", got, tt.wantAdmin)
			}
		})
	}
}

func TestGroupAdminResolver_Cache(t *testing.T) {
	resolver := NewGroupAdminResolver(100 * time.Millisecond)
	groupJID := types.NewJID("120363000000000000", types.GroupServer)

	// Inisialisasi entri cache langsung
	dummyInfo := &types.GroupInfo{
		JID: groupJID,
		Participants: []types.GroupParticipant{
			{
				JID:     types.NewJID("628123456789", types.DefaultUserServer),
				IsAdmin: true,
			},
		},
	}

	resolver.mu.Lock()
	resolver.cache[groupJID.ToNonAD().String()] = &groupAdminCacheEntry{
		info:      dummyInfo,
		expiresAt: time.Now().Add(100 * time.Millisecond),
	}
	resolver.mu.Unlock()

	// 1. Ambil dari cache valid
	info, err := resolver.GetGroupInfo(context.Background(), nil, groupJID)
	if err != nil || info == nil {
		t.Fatalf("Expected cached info, got error: %v", err)
	}
	if len(info.Participants) != 1 {
		t.Errorf("Expected 1 participant from cache, got %d", len(info.Participants))
	}

	// 2. Invalidate cache
	resolver.Invalidate(groupJID)
	resolver.mu.RLock()
	_, found := resolver.cache[groupJID.ToNonAD().String()]
	resolver.mu.RUnlock()
	if found {
		t.Errorf("Expected cache to be invalidated")
	}

	// 3. Simpan lagi dan uji kedaluwarsa (TTL)
	resolver.mu.Lock()
	resolver.cache[groupJID.ToNonAD().String()] = &groupAdminCacheEntry{
		info:      dummyInfo,
		expiresAt: time.Now().Add(50 * time.Millisecond),
	}
	resolver.mu.Unlock()

	time.Sleep(60 * time.Millisecond)

	// Karena client == nil dan cache expired, fallback mengembalikan cache lama yang tersedia
	infoExpiredFallback, err := resolver.GetGroupInfo(context.Background(), nil, groupJID)
	if err != nil || infoExpiredFallback == nil {
		t.Errorf("Expected fallback to stale cache when client is nil, got %v", err)
	}
}
