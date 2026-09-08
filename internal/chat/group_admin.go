package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// groupAdminCacheEntry menyimpan cache data info grup dan waktu kedaluwarsa
type groupAdminCacheEntry struct {
	info      *types.GroupInfo
	expiresAt time.Time
}

// GroupAdminResolver mengelola pengecekan dan caching status admin di grup WhatsApp,
// dengan dukungan penuh untuk resolusi LID (Linked ID) dan nomor telepon (Phone Number).
type GroupAdminResolver struct {
	mu    sync.RWMutex
	ttl   time.Duration
	cache map[string]*groupAdminCacheEntry
}

// NewGroupAdminResolver membuat instance baru GroupAdminResolver dengan TTL cache tertentu
func NewGroupAdminResolver(ttl time.Duration) *GroupAdminResolver {
	if ttl <= 0 {
		ttl = 3 * time.Minute
	}
	return &GroupAdminResolver{
		ttl:   ttl,
		cache: make(map[string]*groupAdminCacheEntry),
	}
}

// CheckAdminStatus adalah fungsi murni untuk memverifikasi apakah pengirim adalah admin grup
// berdasarkan data GroupInfo, JID pengirim, JID alternatif (SenderAlt), dan LID store (opsional).
func CheckAdminStatus(
	ctx context.Context,
	info *types.GroupInfo,
	senderJID, senderAltJID types.JID,
	lidStore store.LIDStore,
) bool {
	if info == nil {
		return false
	}

	// Kumpulkan semua kemungkinan identifier user yang merepresentasikan pengirim pesan
	candidates := make(map[string]bool)

	cleanSender := senderJID.ToNonAD()
	if cleanSender.User != "" {
		candidates[cleanSender.User] = true
	}

	cleanAlt := senderAltJID.ToNonAD()
	if cleanAlt.User != "" {
		candidates[cleanAlt.User] = true
	}

	// Jika sender adalah LID dan lidStore tersedia, coba cari nomor HP aslinya
	if cleanSender.Server == "lid" && lidStore != nil {
		pn, err := lidStore.GetPNForLID(ctx, cleanSender)
		if err == nil && pn.User != "" {
			candidates[pn.User] = true
		}
	}

	// Jika sender adalah nomor HP biasa dan lidStore tersedia, coba cari LID-nya
	if cleanSender.Server == types.DefaultUserServer && lidStore != nil {
		lid, err := lidStore.GetLIDForPN(ctx, cleanSender)
		if err == nil && lid.User != "" {
			candidates[lid.User] = true
		}
	}

	// 1. Periksa apakah pengirim adalah Pemilik Grup (Owner)
	cleanOwner := info.OwnerJID.ToNonAD()
	if cleanOwner.User != "" && candidates[cleanOwner.User] {
		return true
	}
	cleanOwnerPN := info.OwnerPN.ToNonAD()
	if cleanOwnerPN.User != "" && candidates[cleanOwnerPN.User] {
		return true
	}

	// 2. Periksa daftar partisipan grup
	for _, p := range info.Participants {
		if !p.IsAdmin && !p.IsSuperAdmin {
			continue
		}

		// Cocokkan terhadap p.JID
		pJID := p.JID.ToNonAD()
		if pJID.User != "" && candidates[pJID.User] {
			return true
		}

		// Cocokkan terhadap p.PhoneNumber (jika tersedia)
		pPN := p.PhoneNumber.ToNonAD()
		if pPN.User != "" && candidates[pPN.User] {
			return true
		}

		// Cocokkan terhadap p.LID (jika tersedia)
		pLID := p.LID.ToNonAD()
		if pLID.User != "" && candidates[pLID.User] {
			return true
		}
	}

	return false
}

// GetGroupInfo mengambil info grup dari cache jika masih berlaku, atau request ke server WhatsApp
func (r *GroupAdminResolver) GetGroupInfo(ctx context.Context, client *whatsmeow.Client, groupJID types.JID) (*types.GroupInfo, error) {
	key := groupJID.ToNonAD().String()
	now := time.Now()

	r.mu.RLock()
	entry, found := r.cache[key]
	r.mu.RUnlock()

	if found && entry != nil && now.Before(entry.expiresAt) {
		return entry.info, nil
	}

	if client == nil {
		if found && entry != nil && entry.info != nil {
			return entry.info, nil
		}
		return nil, fmt.Errorf("klien whatsmeow belum terinisialisasi")
	}

	info, err := client.GetGroupInfo(ctx, groupJID)
	if err != nil {
		// Fallback cerdas: Jika ada kendala jaringan sesaat, gunakan data cache yang sudah ada
		if found && entry != nil && entry.info != nil {
			return entry.info, nil
		}
		return nil, err
	}

	r.mu.Lock()
	r.cache[key] = &groupAdminCacheEntry{
		info:      info,
		expiresAt: now.Add(r.ttl),
	}
	r.mu.Unlock()

	return info, nil
}

// Invalidate menghapus cache grup tertentu (berguna jika ada perubahan struktur admin grup)
func (r *GroupAdminResolver) Invalidate(groupJID types.JID) {
	key := groupJID.ToNonAD().String()
	r.mu.Lock()
	delete(r.cache, key)
	r.mu.Unlock()
}

// ResolveSenderAdmin memeriksa apakah pengirim pesan merupakan admin atau superadmin di grup
func (r *GroupAdminResolver) ResolveSenderAdmin(
	ctx context.Context,
	client *whatsmeow.Client,
	isGroup bool,
	groupJID, senderJID, senderAltJID types.JID,
) bool {
	if !isGroup {
		return true
	}

	info, err := r.GetGroupInfo(ctx, client, groupJID)
	if err != nil || info == nil {
		return false
	}

	var lidStore store.LIDStore
	if client != nil && client.Store != nil {
		lidStore = client.Store.LIDs
	}

	return CheckAdminStatus(ctx, info, senderJID, senderAltJID, lidStore)
}
