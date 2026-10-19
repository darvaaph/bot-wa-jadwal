/**
 * web/js/app-portal.js — Portal mahasiswa (REVISI Figma): hanya-baca, tanpa akun.
 * Akses kelas diverifikasi ulang oleh backend saat portal dibuka.
 */
function portalApp() {
  return {
    unlocked: localStorage.getItem('portal_pin_ok') === '1',
    pin: '',
    pinError: '',
    view: 'dashboard',
    drawer: false,
    sidebarCollapsed: false,
    pageState: null,
    dashboardLoading: true,
    q: '',
    unreadCount: 0,
    botOnline: false,
    contextAssignments: [],
    contextSwitching: false,
    meCache: null,
    currentUser: null,
    botConnected: false,
    pjMatkul: '',
    pjNavIcon() { return ''; },
    resetPasswordForm: { old_password: '', password: '', confirm: '', showOld: false, showNew: false, showConfirm: false },
    resetPasswordError: '',
    resetPasswordLoading: false,
    hari: 'Senin',
    tugasMatkul: 'Semua',

    ...AsteriskShell.behavior('portal'),

    navSections: [
      {
        title: '', items: [
          { id: 'dashboard', label: 'Beranda', icon: 'space_dashboard' },
          { id: 'jadwal', label: 'Jadwal Kuliah', icon: 'calendar_month' },
          { id: 'tugas', label: 'Tugas', icon: 'assignment', active: ['tugas', 'detail-tugas'] },
          { id: 'materi', label: 'Materi', icon: 'menu_book' }
        ]
      }
    ],

    portalNavIcon(item) {
      const icon = item && item.icon;
      const paths = {
        'space_dashboard': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.36963q-0.55371 0-0.94336-0.38623-0.38623-0.38965-0.38623-0.94336l0-8.08008q0-0.55371 0.38623-0.93994 0.38965-0.38965 0.94336-0.38965l8.08008 0q0.55371 0 0.93994 0.38965 0.38965 0.38965 0.38965 0.93994l0 8.08008q0 0.55371-0.38965 0.94336-0.38623 0.38623-0.93994 0.38623l-8.08008 0z m0-1.32959l3.45557 0 0-8.08008-3.45557 0 0 8.08008z m4.62451 0l3.45557 0 0-4.04004-3.45557 0 0 4.04004z m0-5.20557l3.45557 0 0-2.87451-3.45557 0 0 2.87451z"/></svg>',
        'assignment': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.36963q-0.55029 0-0.93994-0.38965-0.38965-0.38965-0.38965-0.93994l0-8.08008q0-0.55029 0.38623-0.93994 0.38965-0.38965 0.94336-0.38965l2.33447 0q0.21191-0.52979 0.67334-0.84766 0.46484-0.31787 1.03223-0.31787 0.56738 0 1.02881 0.31787 0.46484 0.31787 0.67676 0.84766l2.33447 0q0.55029 0 0.93994 0.38965 0.38965 0.38965 0.38965 0.93994l0 8.08008q0 0.55029-0.38965 0.93994-0.38965 0.38965-0.93994 0.38965l-8.08008 0z m0-1.32959l8.08008 0 0-8.08008-8.08008 0 0 8.08008z m1.75-1.16553l2.86426 0q0.24609 0 0.41357-0.16748 0.16748-0.16748 0.16748-0.41699 0-0.24609-0.16748-0.41358-0.16748-0.16748-0.41357-0.16748l-2.86426 0q-0.24951 0-0.41699 0.16748-0.16748 0.16748-0.16748 0.41358 0 0.24951 0.16748 0.41699 0.16748 0.16748 0.41699 0.16748z m0-2.29004l4.58008 0q0.24951 0 0.41699-0.16748 0.16748-0.16748 0.16748-0.41699 0-0.24951-0.16748-0.41699-0.16748-0.16748-0.41699-0.16748l-4.58008 0q-0.24951 0-0.41699 0.16748-0.16748 0.16748-0.16748 0.41699 0-0.24951 0.16748 0.41699 0.16748 0.16748 0.41699 0.16748z m0-2.29345l4.58008 0q0.24951 0 0.41699-0.16748 0.16748-0.16748 0.16748-0.41358 0-0.24951-0.16748-0.41699-0.16748-0.16748-0.41699-0.16748l-4.58008 0q-0.24951 0-0.41699 0.16748-0.16748 0.16748-0.16748 0.41699 0 0.24609 0.16748 0.41358 0.16748 0.16748 0.41699 0.16748z m2.29004-2.78565q0.20166 0 0.33154-0.12988 0.1333-0.1333 0.1333-0.33496 0-0.20166-0.1333-0.33155-0.12988-0.1333-0.33154-0.1333-0.20166 0-0.33496 0.1333-0.12988 0.12988-0.12988 0.33155 0 0.20166 0.12988 0.33496 0.1333 0.12988 0.33496 0.12988z m-4.04004 8.53467l0-8.08008 0 8.08008z"/></svg>',
        'calendar_month': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.95068q-0.55371 0-0.94336-0.38623-0.38623-0.38965-0.38623-0.93994l0-8.08349q0-0.55029 0.38623-0.93653 0.38965-0.38965 0.94336-0.38965l0.54004 0 0-0.54345q0-0.26318 0.18115-0.44092 0.18115-0.18115 0.44434-0.18115 0.26318 0 0.44092 0.18115 0.18115 0.17773 0.18115 0.44092l0 0.54345 4.50488 0 0-0.54345q0-0.26318 0.18115-0.44092 0.18115-0.18115 0.44434-0.18115 0.26318 0 0.44092 0.18115 0.18115 0.17773 0.18115 0.44092l0 0.54345 0.54004 0q0.55371 0 0.93994 0.38965 0.38965 0.38623 0.38965 0.93653l0 8.08349q0 0.55029-0.38965 0.93994-0.38623 0.38623-0.93994 0.38623l-8.08008 0z m0-1.32617l8.08008 0 0-5.79004-8.08008 0 0 5.79004z m0-6.95898l8.08008 0 0-1.12451-8.08008 0 0 1.12451z m0 0l0-1.12451 0 1.12451z m4.04004 3.52734q-0.25977 0-0.4375-0.17432-0.17432-0.17432-0.17432-0.43408 0-0.25977 0.17432-0.43408 0.17773-0.17773 0.4375-0.17773 0.25977 0 0.43408 0.17773 0.17773 0.17432 0.17774 0.43408 0 0.25977-0.17774 0.43408-0.17432 0.17432-0.43408 0.17432z m-2.33447 0q-0.25977 0-0.43408-0.17432-0.17432-0.17432-0.17432-0.43408 0-0.25977 0.17432-0.43408 0.17432-0.17773 0.43408-0.17773 0.25977 0 0.43408 0.17773 0.17773 0.17432 0.17773 0.43408 0 0.25977-0.17773 0.43408-0.17432 0.17774-0.43408 0.17773z m4.66894 0q-0.25635 0-0.43408-0.17432-0.17773-0.17773-0.17773-0.43408 0-0.25635 0.17773-0.43067 0.17773-0.17773 0.43408-0.17773 0.25635 0 0.43067 0.17773 0.17773 0.17432 0.17773 0.43408 0 0.25977-0.17432 0.43408-0.17432 0.17774-0.43408 0.17773z m-2.33447 2.33447q-0.25977 0-0.4375-0.17773-0.17432-0.17773-0.17432-0.43408 0-0.25635 0.17432-0.43067 0.17773-0.17773 0.4375-0.17773 0.25977 0 0.43408 0.17432 0.17773 0.17773 0.17774 0.43408 0 0.25635-0.17774 0.43408-0.17432 0.17774-0.43408 0.17773z m-2.33447 0q-0.25977 0-0.43408-0.17773-0.17432-0.17773-0.17432-0.43408 0-0.25635 0.17432-0.43067 0.17432-0.17773 0.43408-0.17773 0.25977 0 0.43408 0.17773 0.17773 0.17773 0.17773 0.43408 0 0.25977-0.17773 0.43408-0.17432 0.17774-0.43408 0.17773z m4.66894 0q-0.25635 0-0.43408-0.17773-0.17773-0.17773-0.17773-0.43408 0-0.25635 0.17773-0.43067 0.17773-0.17773 0.43408-0.17773 0.25635 0 0.43067 0.17773 0.17773 0.17773 0.17773 0.43408 0 0.25977-0.17432 0.43408-0.17432 0.17774-0.43408 0.17773z"/></svg>',
        'menu_book': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M3.79053 9.46094q0.68701 0 1.33642 0.15381 0.64941 0.15381 1.28858 0.46142l0-5.7456q-0.59814-0.35205-1.26807-0.52637-0.66992-0.17432-1.35693-0.17432-0.52295 0-1.04248 0.10254-0.51611 0.09912-0.99805 0.3042l0 5.77637q0.50928-0.17432 1.01172-0.26319 0.50586-0.08887 1.02881-0.08886z m3.79394 0.61523q0.63916-0.30762 1.28858-0.46142 0.64941-0.15381 1.33642-0.15381 0.52295 0 1.02539 0.08886 0.50586 0.08887 1.01514 0.26319l0-5.77637q-0.48193-0.20508-1.00146-0.3042-0.51611-0.10254-1.03907-0.10254-0.68701 0-1.35693 0.17432-0.66992 0.17432-1.26807 0.52637l0 5.7456z m-0.57422 1.53125q-0.2085 0-0.3999-0.05127-0.19141-0.05127-0.35547-0.14355-0.56738-0.33154-1.18945-0.50586-0.62207-0.17432-1.2749-0.17432-0.59814 0-1.1792 0.16406-0.57764 0.16065-1.11084 0.44776-0.35547 0.18457-0.69043-0.01709-0.33496-0.20166-0.33496-0.59131l0-7.02393q0-0.2085 0.09912-0.39306 0.09912-0.18457 0.29736-0.27686 0.68018-0.34863 1.41162-0.51611 0.73145-0.1709 1.49707-0.1709 0.85449 0 1.66455 0.21875 0.81348 0.21875 1.55518 0.6665 0.74512-0.44434 1.55518-0.66308 0.81006-0.22217 1.66455-0.22217 0.76563 0 1.49707 0.1709 0.73145 0.16748 1.41162 0.51611 0.19824 0.09229 0.29736 0.27686 0.09912 0.18457 0.09912 0.39306l0 7.07862q0 0.37256-0.33496 0.55713-0.33154 0.18457-0.69043-0.00342-0.5332-0.28711-1.11426-0.44776-0.57764-0.16406-1.17578-0.16406-0.64941 0-1.26465 0.17774-0.61523 0.17432-1.17578 0.50244-0.16406 0.09229-0.35888 0.14355-0.19141 0.05127-0.39991 0.05127z m1.15528-6.36767q0-0.12646 0.09228-0.25977 0.09229-0.13672 0.2085-0.18115 0.42383-0.14697 0.84765-0.21875 0.42725-0.0752 0.89551-0.0752 0.29053 0 0.57422 0.0376 0.28369 0.03418 0.56055 0.09229 0.1333 0.03076 0.22558 0.15039 0.0957 0.11621 0.09571 0.26318 0 0.24609-0.16065 0.3623-0.16064 0.11279-0.40673 0.05469-0.2085-0.04102-0.43067-0.06152-0.22217-0.02393-0.45801-0.02393-0.37939 0-0.7417 0.07178-0.3623 0.06836-0.69726 0.18799-0.26318 0.09912-0.43408-0.01709-0.1709-0.11621-0.1709-0.38281z m0 3.20947q0-0.12646 0.09228-0.25977 0.09229-0.13672 0.2085-0.18115 0.42383-0.14697 0.84765-0.22217 0.42725-0.0752 0.89551-0.07519 0.29053 0 0.57422 0.0376 0.28369 0.0376 0.56055 0.0957 0.1333 0.03076 0.22558 0.14697 0.0957 0.11621 0.09571 0.26318 0 0.24951-0.16065 0.36573-0.16064 0.11279-0.40673 0.05468-0.2085-0.04443-0.43067-0.06494-0.22217-0.02393-0.45801-0.02392-0.37939 0-0.7417 0.06494-0.3623 0.06494-0.69726 0.18115-0.26318 0.10254-0.43408-0.00683-0.1709-0.11279-0.1709-0.37598z m0-1.60303q0-0.12988 0.09228-0.26318 0.09229-0.13672 0.2085-0.18115 0.42383-0.14355 0.84765-0.21875 0.42725-0.0752 0.89551-0.0752 0.29053 0 0.57422 0.0376 0.28369 0.0376 0.56055 0.0957 0.1333 0.02734 0.22558 0.14697 0.0957 0.11621 0.09571 0.26319 0 0.24951-0.16065 0.36572-0.16064 0.11279-0.40673 0.05469-0.2085-0.04443-0.43067-0.06494-0.22217-0.02393-0.45801-0.02393-0.37939 0-0.7417 0.07178-0.3623 0.07178-0.69726 0.18799-0.26318 0.10254-0.43408-0.01368-0.1709-0.11963-0.1709-0.38281z"/></svg>',
        'home': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M2.95996 12.36963q-0.55371 0-0.94336-0.38623-0.38623-0.38965-0.38623-0.94336l0-8.08008q0-0.55371 0.38623-0.93994 0.38965-0.38965 0.94336-0.38965l8.08008 0q0.55371 0 0.93994 0.38965 0.38965 0.38965 0.38965 0.93994l0 8.08008q0 0.55371-0.38965 0.94336-0.38623 0.38623-0.93994 0.38623l-8.08008 0z m0-1.32959l3.45557 0 0-8.08008-3.45557 0 0 8.08008z m4.62451 0l3.45557 0 0-4.04004-3.45557 0 0 4.04004z m0-5.20557l3.45557 0 0-2.87451-3.45557 0 0 2.87451z"/></svg>',
        'folder': '<svg width="20" height="20" viewBox="0 0 14 14" fill="currentColor" class="w-5 h-5 shrink-0" aria-hidden="true" focusable="false"><path d="M3.79053 9.46094q0.68701 0 1.33642 0.15381 0.64941 0.15381 1.28858 0.46142l0-5.7456q-0.59814-0.35205-1.26807-0.52637-0.66992-0.17432-1.35693-0.17432-0.52295 0-1.04248 0.10254-0.51611 0.09912-0.99805 0.3042l0 5.77637q0.50928-0.17432 1.01172-0.26319 0.50586-0.08887 1.02881-0.08886z m3.79394 0.61523q0.63916-0.30762 1.28858-0.46142 0.64941-0.15381 1.33642-0.15381 0.52295 0 1.02539 0.08886 0.50586 0.08887 1.01514 0.26319l0-5.77637q-0.48193-0.20508-1.00146-0.3042-0.51611-0.10254-1.03907-0.10254-0.68701 0-1.35693 0.17432-0.66992 0.17432-1.26807 0.52637l0 5.7456z m-0.57422 1.53125q-0.2085 0-0.3999-0.05127-0.19141-0.05127-0.35547-0.14355-0.56738-0.33154-1.18945-0.50586-0.62207-0.17432-1.2749-0.17432-0.59814 0-1.1792 0.16406-0.57764 0.16065-1.11084 0.44776-0.35547 0.18457-0.69043-0.01709-0.33496-0.20166-0.33496-0.59131l0-7.02393q0-0.2085 0.09912-0.39306 0.09912-0.18457 0.29736-0.27686 0.68018-0.34863 1.41162-0.51611 0.73145-0.1709 1.49707-0.1709 0.85449 0 1.66455 0.21875 0.81348 0.21875 1.55518 0.6665 0.74512-0.44434 1.55518-0.66308 0.81006-0.22217 1.66455-0.22217 0.76563 0 1.49707 0.1709 0.73145 0.16748 1.41162 0.51611 0.19824 0.09229 0.29736 0.27686 0.09912 0.18457 0.09912 0.39306l0 7.07862q0 0.37256-0.33496 0.55713-0.33154 0.18457-0.69043-0.00342-0.5332-0.28711-1.11426-0.44776-0.57764-0.16406-1.17578-0.16406-0.64941 0-1.26465 0.17774-0.61523 0.17432-1.17578 0.50244-0.16406 0.09229-0.35888 0.14355-0.19141 0.05127-0.39991 0.05127z m1.15528-6.36767q0-0.12646 0.09228-0.25977 0.09229-0.13672 0.2085-0.18115 0.42383-0.14697 0.84765-0.21875 0.42725-0.0752 0.89551-0.0752 0.29053 0 0.57422 0.0376 0.28369 0.03418 0.56055 0.09229 0.1333 0.03076 0.22558 0.15039 0.0957 0.11621 0.09571 0.26318 0 0.24609-0.16065 0.3623-0.16064 0.11279-0.40673 0.05469-0.2085-0.04102-0.43067-0.06152-0.22217-0.02393-0.45801-0.02393-0.37939 0-0.7417 0.07178-0.3623 0.06836-0.69726 0.18799-0.26318 0.09912-0.43408-0.01709-0.1709-0.11621-0.1709-0.38281z m0 3.20947q0-0.12646 0.09228-0.25977 0.09229-0.13672 0.2085-0.18115 0.42383-0.14697 0.84765-0.22217 0.42725-0.0752 0.89551-0.07519 0.29053 0 0.57422 0.0376 0.28369 0.0376 0.56055 0.0957 0.1333 0.03076 0.22558 0.14697 0.0957 0.11621 0.09571 0.26318 0 0.24951-0.16065 0.36573-0.16064 0.11279-0.40673 0.05468-0.2085-0.04443-0.43067-0.06494-0.22217-0.02393-0.45801-0.02392-0.37939 0-0.7417 0.06494-0.3623 0.06494-0.69726 0.18115-0.26318 0.10254-0.43408-0.00683-0.1709-0.11279-0.1709-0.37598z m0-1.60303q0-0.12988 0.09228-0.26318 0.09229-0.13672 0.2085-0.18115 0.42383-0.14355 0.84765-0.21875 0.42725-0.0752 0.89551-0.0752 0.29053 0 0.57422 0.0376 0.28369 0.0376 0.56055 0.0957 0.1333 0.02734 0.22558 0.14697 0.0957 0.11621 0.09571 0.26319 0 0.24951-0.16065 0.36572-0.16064 0.11279-0.40673 0.05469-0.2085-0.04443-0.43067-0.06494-0.22217-0.02393-0.45801-0.02393-0.37939 0-0.7417 0.07178-0.3623 0.07178-0.69726 0.18799-0.26318 0.10254-0.43408-0.01368-0.1709-0.11963-0.1709-0.38281z"/></svg>'
      };
      return paths[icon] || paths['space_dashboard'] || '';
    },

    todayName: 'Senin',
    todayFull: '',
    semesterLabel: '',
    selectedClass: '',
    selectedProdi: 'D4',
    selectedSemester: '3',
    selectedAbjad: 'A',
    classList: [],
    classModalOpen: false,
    comboboxOpen: false,
    classQuery: '',
    classFilterProdi: 'ALL',
    gateStep: 'pin',
    checkingAccess: false,
    fullSchedule: [],
    jadwalEfektif: [],
    jadwalCacheHariIni: [],
    jadwalPekan: { Senin: [], Selasa: [], Rabu: [], Kamis: [], Jumat: [], Sabtu: [] },
    weekOffset: 0,
    modeTampilan: 'kalender',
    listDayFilter: 'all',
    schedSearch: '',
    mobileJadwalMode: 'per_hari',
    detailModalOpen: false,
    detailSesi: null,
    sheetSesiOffsetY: 0,
    sheetSesiDragging: false,
    sheetSesiClosing: false,
    sheetSesiResetting: false,
    sheetSesiStartY: 0,
    modalRiwayatOpen: false,
    sheetRiwayatOffsetY: 0,
    sheetRiwayatDragging: false,
    sheetRiwayatClosing: false,
    sheetRiwayatResetting: false,
    sheetRiwayatStartY: 0,
    pekanHariListDesktop: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'],
    pekanHariListMobile: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'],
    jadwalLoading: false,
    jadwalError: '',
    tasks: [],
    tugasGrup: { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] },
    tugasTab: 'mendatang',
    tugasMatkul: 'Semua',
    tugasLoading: false,
    tugasError: '',
    materiList: [],
    materiLoading: false,
    materiError: '',
    filterMateriMatkul: 'SEMUA',
    filterMateriSesi: 'SEMUA',
    searchMateriQuery: '',
    sortMateri: 'terbaru',
    detailMateriModalOpen: false,
    selectedMateri: null,
    sheetMateriOffsetY: 0,
    sheetMateriDragging: false,
    sheetMateriClosing: false,
    sheetMateriResetting: false,
    sheetMateriStartY: 0,
    courses: [], coursesLoading: false, coursesError: '', selectedCourse: null,
    semesterList: [],
    semesterLoading: false,
    archiveSelected: null,
    archiveLoading: false,
    archiveError: '',
    archiveData: { schedule: [], tasks: [], materials: [], changes: [] },
    perubahanList: [],
    perubahanLoading: false,
    detailTugas: {},
    detailMateri: [],

    toast: { show: false, message: '', timer: null },

    get todayList() {
      return this.jadwalEfektifHariIni;
    },

    // Tanggal ISO tiap hari Senin–Minggu pada pekan terpilih (berdasarkan weekOffset, zona WIB).
    get tanggalPekan() {
      const out = {};
      try {
        const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
        const nowWib = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
        const dow = (nowWib.getDay() + 6) % 7;
        const monday = new Date(nowWib);
        monday.setDate(nowWib.getDate() - dow + (this.weekOffset * 7));
        names.forEach((n, i) => {
          const d = new Date(monday);
          d.setDate(monday.getDate() + i);
          const pad = (x) => String(x).padStart(2, '0');
          out[n] = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
        });
      } catch (e) { }
      return out;
    },

    get tanggalPekanFormatted() {
      const out = {};
      const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
      const names = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
      names.forEach(n => {
        const iso = this.tanggalPekan[n];
        if (!iso) { out[n] = ''; return; }
        const parts = iso.split('-');
        const day = parseInt(parts[2], 10);
        const m = parseInt(parts[1], 10) - 1;
        out[n] = `${day} ${months[m] || ''}`;
      });
      return out;
    },

    get pekanNum() {
      try {
        if (this.semesterList && this.semesterList.length) {
          const active = this.semesterList.find(s => String(s.status || '').toUpperCase() === 'ACTIVE');
          if (active && active.starts_on) {
            const start = new Date(active.starts_on);
            const now = new Date();
            const diffDays = Math.floor((now - start) / (24 * 3600 * 1000));
            const w = Math.floor(diffDays / 7) + 1;
            if (w >= 1 && w <= 20) return w;
          }
        }
      } catch (e) { }
      return 6; // fallback sesuai Figma: Pekan Ke-6
    },

    get pekanAktifNum() {
      return Math.max(1, this.pekanNum + (this.weekOffset || 0));
    },

    get rentangPekanLabel() {
      try {
        const sen = this.tanggalPekan['Senin'];
        const min = this.tanggalPekan['Minggu'];
        if (!sen || !min) return 'Pekan Ini';
        const d1 = new Date(sen + 'T00:00:00+07:00');
        const d2 = new Date(min + 'T00:00:00+07:00');
        const mNames = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
        if (d1.getMonth() === d2.getMonth() && d1.getFullYear() === d2.getFullYear()) {
          return `${d1.getDate()} – ${d2.getDate()} ${mNames[d1.getMonth()]} ${d1.getFullYear()}`;
        } else if (d1.getFullYear() === d2.getFullYear()) {
          return `${d1.getDate()} ${mNames[d1.getMonth()]} – ${d2.getDate()} ${mNames[d2.getMonth()]} ${d2.getFullYear()}`;
        }
        return `${d1.getDate()} ${mNames[d1.getMonth()]} ${d1.getFullYear()} – ${d2.getDate()} ${mNames[d2.getMonth()]} ${d2.getFullYear()}`;
      } catch (e) {
        return 'Pekan Ini';
      }
    },

    get rentangPekanSingkatLabel() {
      try {
        const sen = this.tanggalPekan['Senin'];
        const min = this.tanggalPekan['Minggu'];
        if (!sen || !min) return `Pekan ${this.pekanAktifNum}`;
        const d1 = new Date(sen + 'T00:00:00+07:00');
        const d2 = new Date(min + 'T00:00:00+07:00');
        const mShort = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        if (d1.getMonth() === d2.getMonth()) {
          return `${d1.getDate()} - ${d2.getDate()} ${mShort[d1.getMonth()]} (Pekan ${this.pekanAktifNum})`;
        }
        return `${d1.getDate()} ${mShort[d1.getMonth()]} - ${d2.getDate()} ${mShort[d2.getMonth()]} (Pekan ${this.pekanAktifNum})`;
      } catch (e) {
        return `Pekan ${this.pekanAktifNum}`;
      }
    },

    get tanggalHariTerpilihFormat() {
      try {
        const iso = this.tanggalPekan[this.hari];
        if (!iso) return this.hari;
        const d = new Date(iso + 'T00:00:00+07:00');
        const mNames = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
        return `${this.hari}, ${d.getDate()} ${mNames[d.getMonth()]} ${d.getFullYear()}`;
      } catch (e) {
        return this.hari;
      }
    },

    get activeTasksCount() {
      return (this.tasks || []).length;
    },

    get hasSaturdayClass() {
      return Array.isArray(this.jadwalPekan['Sabtu']) && this.jadwalPekan['Sabtu'].length > 0;
    },

    get pekanHariListAgenda() {
      if (this.hasSaturdayClass) {
        return ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      }
      return ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat'];
    },

    formatTanggalLengkap(h) {
      try {
        const iso = this.tanggalPekan[h];
        if (!iso) return h;
        const d = new Date(iso + 'T00:00:00+07:00');
        const mNames = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
        return `${h}, ${d.getDate()} ${mNames[d.getMonth()]} ${d.getFullYear()}`;
      } catch (e) {
        return h;
      }
    },

    portalDaySessions(d) {
      const list = this.jadwalPekan[d] || [];
      const q = (this.schedSearch || '').toLowerCase().trim();
      if (!q) return list;
      return list.filter(s => {
        const hay = [s.matkul, s.dosen, s.ruang].filter(Boolean).join(' ').toLowerCase();
        return hay.includes(q);
      });
    },

    allSessionsCount() {
      return this.pekanHariListAgenda.reduce((acc, d) => acc + this.portalDaySessions(d).length, 0);
    },

    listDaysToShow() {
      const days = this.pekanHariListAgenda;
      if (this.listDayFilter === 'all') {
        const active = days.filter(d => this.portalDaySessions(d).length > 0);
        return active.length > 0 ? active : days;
      }
      return days.filter(d => d === this.listDayFilter);
    },

    portalSesiType(s) {
      if (!s) return 'TEORI';
      if (s.kind === 'PENGGANTI') return 'PENGGANTI';
      const m = String(s.matkul || '').toLowerCase();
      const r = String(s.ruang || '').toLowerCase();
      const act = String(s.activityType || '').toUpperCase();
      if (act === 'PRACTICE' || act === 'PRAKTIKUM' || act === 'LAB' || m.includes('praktik') || r.includes('lab')) return 'PRAKTIKUM';
      return 'TEORI';
    },

    portalSesiBadgeClass(s) {
      const type = this.portalSesiType(s);
      if (type === 'PENGGANTI') {
        return 'bg-amber-50 text-amber-800 border-amber-200/80';
      }
      if (type === 'PRAKTIKUM') {
        return 'bg-purple-50 text-purple-700 border-purple-200/80';
      }
      return 'bg-blue-50 text-blue-700 border-blue-200/80';
    },

    get jadwalEfektifHariIni() {
      return (this.jadwalCacheHariIni || []).slice().sort((a, b) =>
        String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    get currentSession() {
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const pad = n => String(n).padStart(2, '0');
      const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`;
      return (this.jadwalEfektifHariIni || []).find(s => {
        if (!s.timeStart || !s.timeEnd) return false;
        return nowHM >= s.timeStart && nowHM <= s.timeEnd;
      }) || null;
    },

    get nextSession() {
      const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
      const pad = n => String(n).padStart(2, '0');
      const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`;
      return (this.jadwalEfektifHariIni || [])
        .filter(s => s.timeStart && s.timeStart > nowHM)
        .slice()
        .sort((a, b) => String(a.timeStart || '').localeCompare(String(b.timeStart || '')))[0] || null;
    },

    get withUrgency() {
      const rank = { mendesak: 0, mendekati: 1, aman: 2 };
      return [...this.tasks].sort((a, b) => (rank[a.urgency] ?? 2) - (rank[b.urgency] ?? 2));
    },

    get urgent3() { return this.withUrgency.slice(0, 3); },
    get nearCount() { return this.tasks.filter(t => t.urgency !== 'aman').length; },

    get matkulOpts() {
      const dariTugas = (this.tasks || []).map(t => t.matkul).filter(Boolean);
      const dariJadwal = (this.fullSchedule || []).map(s => s.matkul).filter(Boolean);
      return Array.from(new Set([...dariTugas, ...dariJadwal]));
    },

    get tugasTabList() {
      const list = this.tugasGrup[this.tugasTab] || [];
      return list.filter(t => (this.tugasMatkul === 'Semua' || t.matkul === this.tugasMatkul)
        && this.shellMatchesSearch('tugas', [t.title, t.deskripsi, t.instructions, t.matkul]));
    },

    get jadwalHari() {
      return (this.jadwalEfektif || []).filter(s => this.shellMatchesSearch('jadwal', [s.matkul, s.dosen, s.ruang])).sort((a, b) =>
        String(a.timeStart || '').localeCompare(String(b.timeStart || '')));
    },

    getSesiLiveStatus(s, dayName) {
      if (!s) return { label: 'Akan Datang', class: 'bg-[#EFF6FF] text-[#1D4ED8]' };
      const now = new Date();
      const todayIndo = now.toLocaleDateString('id-ID', { weekday: 'long', timeZone: 'Asia/Jakarta' });
      const hari = s.hari || dayName || this.hari;
      if (hari !== todayIndo || (this.weekOffset !== undefined && this.weekOffset !== 0)) {
        return { label: 'Akan Datang', class: 'bg-[#EFF6FF] text-[#1D4ED8]' };
      }
      const timeParts = new Intl.DateTimeFormat('en-GB', {
        timeZone: 'Asia/Jakarta',
        hour: 'numeric',
        minute: 'numeric',
        hour12: false
      }).formatToParts(now);
      const h = parseInt(timeParts.find(p => p.type === 'hour')?.value || '0', 10);
      const m = parseInt(timeParts.find(p => p.type === 'minute')?.value || '0', 10);
      const nowMin = h * 60 + m;

      const toMins = (t) => {
        if (!t) return -1;
        const [hh, mm] = String(t).replace('.', ':').split(':').map(Number);
        return (isNaN(hh) || isNaN(mm)) ? -1 : hh * 60 + mm;
      };

      const start = toMins(s.timeStart);
      const end = toMins(s.timeEnd);
      if (start >= 0 && end >= 0) {
        if (nowMin >= start && nowMin <= end) {
          return { label: '● Sedang Berlangsung Saat Ini', class: 'bg-[#DCFCE7] text-[#15803D]' };
        } else if (nowMin > end) {
          return { label: 'Selesai Hari Ini', class: 'bg-[#F1F5F9] text-[#64748B]' };
        }
      }
      return { label: 'Akan Datang Hari Ini', class: 'bg-[#EFF6FF] text-[#1D4ED8]' };
    },

    isSesiOngoing(s, dayName) {
      if (!s) return false;
      const now = new Date();
      const todayIndo = now.toLocaleDateString('id-ID', { weekday: 'long', timeZone: 'Asia/Jakarta' });
      const hari = s.hari || dayName || this.hari;
      if (hari !== todayIndo || (this.weekOffset !== undefined && this.weekOffset !== 0)) {
        return false;
      }
      const timeParts = new Intl.DateTimeFormat('en-GB', {
        timeZone: 'Asia/Jakarta',
        hour: 'numeric',
        minute: 'numeric',
        hour12: false
      }).formatToParts(now);
      const h = parseInt(timeParts.find(p => p.type === 'hour')?.value || '0', 10);
      const m = parseInt(timeParts.find(p => p.type === 'minute')?.value || '0', 10);
      const nowMin = h * 60 + m;

      const toMins = (t) => {
        if (!t) return -1;
        const [hh, mm] = String(t).replace('.', ':').split(':').map(Number);
        return (isNaN(hh) || isNaN(mm)) ? -1 : hh * 60 + mm;
      };

      const start = toMins(s.timeStart);
      const end = toMins(s.timeEnd);
      if (start < 0 || end < 0) return false;
      return nowMin >= start && nowMin <= end;
    },

    get perubahanTerbaru() { return (this.perubahanList || []).slice(0, 3); },

    async bukaArsipSemester(semester) {
      if (!semester || String(semester.status || '').toUpperCase() !== 'ARCHIVED') return;
      this.archiveSelected = semester;
      this.archiveLoading = true;
      this.archiveError = '';
      this.archiveData = { schedule: [], tasks: [], materials: [], changes: [] };
      try {
        const start = new Date(`${semester.starts_on}T00:00:00+07:00`);
        const end = new Date(`${semester.ends_on}T23:59:59+07:00`);
        const days = [];
        if (!Number.isNaN(start.getTime())) {
          for (let i = 0; i < 7; i++) {
            const d = new Date(start); d.setDate(start.getDate() + i);
            if (d > end) break;
            days.push(d.toISOString().slice(0, 10));
          }
        }
        const [schedules, tasks, materials, changes] = await Promise.all([
          Promise.all(days.map(d => API.getPortalSchedule(this.selectedClassSlug, d, semester.id))),
          API.getPortalTasks(this.selectedClassSlug, '', semester.id),
          API.getPortalMaterials(this.selectedClassSlug, semester.id),
          API.getChanges(this.selectedClassSlug, semester.id)
        ]);
        this.archiveData = {
          schedule: schedules.flatMap(x => x && Array.isArray(x.items) ? x.items : []),
          tasks: tasks || [], materials: materials || [], changes: changes || []
        };
      } catch (err) {
        this.archiveError = err.message || 'Arsip semester belum dapat dimuat.';
      } finally { this.archiveLoading = false; }
    },

    tutupArsipSemester() {
      this.archiveSelected = null;
      this.archiveError = '';
    },

    badgeJadwal(kind) {
      const k = String(kind || '').toUpperCase();
      if (k === 'PENGGANTI') return { label: 'Kelas Pengganti', cls: 'bg-amber-50 text-amber-900 border-amber-300', icon: 'swap_horiz' };
      if (k === 'TAMBAHAN') return { label: 'Kelas Tambahan', cls: 'bg-primary-soft text-primary border-primary/30', icon: 'add_circle' };
      if (k === 'LIBUR') return { label: 'Diliburkan', cls: 'bg-red-50 text-red-700 border-red-200', icon: 'event_busy' };
      if (k === 'DIBATALKAN') return { label: 'Sesi Dibatalkan', cls: 'bg-red-50 text-red-700 border-red-200', icon: 'cancel' };
      return { label: 'Pola Jadwal', cls: 'bg-success text-green-800 border-green-200', icon: 'event' };
    },

    fmtDeadlineID(iso) { return API.fmtDeadlineID(iso); },
    deadlineBadge(iso, completedAt) { return API.deadlineBadge(iso, completedAt); },

    fmtDeadlineDetailed(iso) {
      if (!iso) return '-';
      try {
        const d = new Date(iso);
        if (isNaN(d.getTime())) return iso;
        const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        const dayName = days[d.getDay()];
        const day = d.getDate();
        const m = months[d.getMonth()];
        const y = d.getFullYear();
        const hh = String(d.getHours()).padStart(2, '0');
        const mm = String(d.getMinutes()).padStart(2, '0');
        return `${dayName}, ${day} ${m} ${y} · ${hh}:${mm} WIB`;
      } catch (e) {
        return iso;
      }
    },

    deadlineRelative(iso) {
      if (!iso) return '-';
      try {
        const target = new Date(iso).getTime();
        const now = Date.now();
        const diffMs = target - now;
        const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
        const diffDays = Math.floor(diffHours / 24);

        if (diffMs < 0) {
          const absHours = Math.abs(diffHours);
          if (absHours < 24) return `Terlewat ${absHours} jam`;
          const absDays = Math.abs(diffDays);
          return `Terlewat ${absDays} hari`;
        }
        if (diffHours < 1) {
          const diffMinutes = Math.floor(diffMs / (1000 * 60));
          return `Sisa ${Math.max(1, diffMinutes)} menit`;
        }
        if (diffHours < 24) return `Sisa ${diffHours} jam`;
        if (diffDays === 1) return 'Besok';
        return `Sisa ${diffDays} hari`;
      } catch (e) {
        return '-';
      }
    },

    urgencyStyle(iso) {
      if (!iso) return { bg: 'bg-[#F1F5F9]', text: 'text-[#475569]', dot: 'bg-[#94A3B8]' };
      try {
        const target = new Date(iso).getTime();
        const now = Date.now();
        const diffMs = target - now;
        const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

        if (diffMs < 0) {
          return { bg: 'bg-[#FEE2E2]', text: 'text-[#991B1B]', dot: 'bg-[#DC2626]' };
        }
        if (diffHours <= 24) {
          return { bg: 'bg-[#FEF2F2]', text: 'text-[#DC2626]', dot: 'bg-[#EF4444]' };
        }
        if (diffHours <= 72) {
          return { bg: 'bg-[#FEF3C7]', text: 'text-[#92400E]', dot: 'bg-[#F59E0B]' };
        }
        return { bg: 'bg-[#DCFCE7]', text: 'text-[#15803D]', dot: 'bg-[#16A34A]' };
      } catch (e) {
        return { bg: 'bg-[#F1F5F9]', text: 'text-[#475569]', dot: 'bg-[#94A3B8]' };
      }
    },

    dashboardPartialsLoaded: false,

    async ensureDashboardPartials() {
      if (this.dashboardPartialsLoaded) return;
      await this.loadPartials([
        ['portal-dashboard', '/partials/portal/view-dashboard.html'],
        ['portal-tugas', '/partials/portal/view-tugas.html'],
        ['portal-jadwal', '/partials/portal/view-jadwal.html'],
        ['portal-materi', '/partials/portal/view-materi.html'],
        ['portal-courses', '/partials/portal/view-courses.html'],
        ['portal-perubahan', '/partials/portal/view-perubahan.html'],
        ['portal-arsip', '/partials/portal/view-arsip.html']
      ]);
      this.dashboardPartialsLoaded = true;
    },

    async initPortal() {
      try { this.sidebarCollapsed = localStorage.getItem('asterisk:sidebar:collapsed') === '1'; } catch (e) { }
      window.addEventListener('offline', () => { this.showPageError('offline'); });
      window.addEventListener('online', () => { if (this.pageState && this.pageState.status === 'offline') window.location.reload(); });
      await this.loadClasses();
      await this.loadPartials([['portal-gate', '/partials/portal/gate.html']]);
      await AsteriskShell.mount('portal', ['dashboard', 'tugas', 'jadwal', 'materi', 'courses', 'perubahan', 'arsip']);

      await this.masukKelas();
    },

    async loadPartials(slots) {
      // Alpine v3 auto-init node baru via MutationObserver; jangan initTree manual (render ganda).
      await Promise.all(slots.map(async ([id, url]) => {
        try {
          const res = await fetch(url + '?v=' + encodeURIComponent(AsteriskShell.version), { cache: 'no-store' });
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          const el = document.getElementById(id);
          if (el) {
            el.innerHTML = await res.text();
          }
        } catch (err) {
          console.error(`Gagal memuat ${url}:`, err);
        }
      }));
    },

    updateClock() {
      try {
        const fmt = new Intl.DateTimeFormat('id-ID', {
          timeZone: 'Asia/Jakarta', weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
        });
        const parts = fmt.formatToParts(new Date());
        const get = (t) => { const p = parts.find(x => x.type === t); return p ? p.value : ''; };
        let day = get('weekday') || 'Senin';
        day = day.charAt(0).toUpperCase() + day.slice(1);
        this.todayName = day;
        this.todayFull = `${day}, ${get('day')} ${get('month')} ${get('year')}`;
      } catch (e) {
        this.todayName = 'Senin';
        this.todayFull = 'Senin';
      }
    },

    get selectedClassSlug() {
      return (this.selectedClass || 'd4-ti-2024-a').toLowerCase().replace(/\s+/g, '-');
    },

    loadingPin: false,

    async masukKelas() {
      if (!this.selectedClass) {
        this.showToast('Pilih kelas terlebih dahulu.');
        return;
      }
      this.checkingAccess = true;
      this.pinError = '';

      try {
        const slug = this.selectedClassSlug;
        const savedToken = localStorage.getItem('portal_token');
        const savedClass = localStorage.getItem('portal_class');
        const tokenToSend = (savedClass === slug && savedToken) ? savedToken : '';

        // Cek apakah kelas ini membutuhkan kode akses v1
        const res = await fetch('/api/v1/portal/' + encodeURIComponent(slug) + '/summary', {
          credentials: 'same-origin',
          headers: tokenToSend ? { 'X-Portal-Token': tokenToSend } : {}
        });

        if (res.ok) {
          // Akses langsung diizinkan (mode LINK atau sesi token valid)
          localStorage.setItem('portal_pin_ok', '1');
          localStorage.setItem('portal_class', slug);
          this.unlocked = true;
          this.gateStep = 'pin';
          this.showToast(`Selamat datang di Portal ${this.selectedClass}!`);
          await this.ensureDashboardPartials();
          this.updateClock();
          this.searchMateriQuery = '';
          await Promise.all([this.loadSchedule(), this.loadJadwalEfektif(), this.loadTugasPortal(), this.loadMateri(), this.loadPerubahan(), this.loadSemester()]);
          this.dashboardLoading = false;
          return;
        }

        if (res.status === 401) {
          // Kelas ini memerlukan PIN (mode CODE)
          this.gateStep = 'pin';
          this.pin = '';
          return;
        }

        // Jika kelas tidak dikenal portal (404): jangan buka; tampilkan error jujur.
        // Status lain (500, dsb): masalah server/koneksi, juga jangan buka diam-diam.
        this.showToast(res.status === 404
          ? 'Kelas tidak terdaftar di portal. Periksa kode kelas atau hubungi KM.'
          : 'Portal belum dapat diakses. Periksa koneksi lalu coba lagi.');
        return;
      } catch (err) {
        // Fallback jika offline/error: tetap terkunci agar tidak tampil portal kosong.
        this.showToast('Tidak dapat terhubung ke portal. Periksa koneksi lalu coba lagi.');
        return;
      } finally {
        this.checkingAccess = false;
        this.dashboardLoading = false;
      }
    },

    kembaliKePilihKelas() {
      this.classModalOpen = true;
    },

    async pilihKelasModal(cls) {
      this.selectedClass = cls;
      this.pin = '';
      this.pinError = '';
      this.classModalOpen = false;
      const slug = (cls || '').toLowerCase().replace(/\s+/g, '-');
      try {
        window.history.replaceState({}, '', '/c/' + encodeURIComponent(slug));
      } catch (e) { }
      await this.masukKelas();
    },

    async bukaPortal() {
      const code = this.pin.trim();
      if (code.length < 6 || code.length > 128) {
        this.pinError = 'Kode akses harus terdiri dari 6 sampai 128 karakter.';
        return;
      }
      this.pinError = '';
      this.loadingPin = true;

      try {
        const slug = this.selectedClassSlug;
        const res = await API.verifyPortalCode(slug, code);

        if (res && res.portal_token) {
          localStorage.setItem('portal_token', res.portal_token);
        }
        localStorage.setItem('portal_pin_ok', '1');
        localStorage.setItem('portal_class', slug);
        localStorage.setItem('portal_pin_at', String(Date.now()));
        this.unlocked = true;
        this.gateStep = 'pin';
        this.showToast('Kode akses terverifikasi. Selamat datang di Portal Kelas!');

        await this.ensureDashboardPartials();
        this.updateClock();
        this.searchMateriQuery = '';
        await Promise.all([this.loadSchedule(), this.loadJadwalEfektif(), this.loadTugasPortal(), this.loadMateri(), this.loadPerubahan(), this.loadSemester()]);
        this.dashboardLoading = false;
      } catch (err) {
        if (err && err.code === 'RATE_LIMITED') {
          this.pinError = 'Terlalu banyak percobaan salah. Silakan tunggu 15 menit.';
        } else {
          this.pinError = 'Kode akses salah atau telah diperbarui oleh Ketua Murid. Silakan minta kode terbaru di grup kelas.';
        }
        this.showToast(this.pinError);
      } finally {
        this.loadingPin = false;
        this.dashboardLoading = false;
      }
    },

    kunciPortal() {
      localStorage.removeItem('portal_pin_ok');
      localStorage.removeItem('portal_token');
      localStorage.removeItem('portal_class');
      this.unlocked = false;
      this.gateStep = 'pin';
      this.pin = '';
      this.pinError = '';
      this.view = 'dashboard';
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      try { localStorage.setItem('asterisk:sidebar:collapsed', this.sidebarCollapsed ? '1' : '0'); } catch (e) { }
    },

    knownViews: ['dashboard', 'tugas', 'detail-tugas', 'jadwal', 'courses', 'course-detail', 'materi', 'detail-materi', 'perubahan', 'arsip'],

    showPageError(status) {
      this.pageState = { status: status };
      this.drawer = false;
      this.view = '__error';
      window.scrollTo({ top: 0 });
    },

    async pilihKelas(kelas) {
      if (!kelas || kelas === this.selectedClass) return;
      this.selectedClass = kelas;
      this.syncSegmentsFromClass();
      this.gateStep = 'pin';
      this.pin = '';
      this.pinError = '';

      if (this.unlocked) {
        const savedClass = localStorage.getItem('portal_class');
        this.unlocked = (savedClass === this.selectedClassSlug && localStorage.getItem('portal_pin_ok') === '1');
        if (this.unlocked) {
          await this.ensureDashboardPartials();
          this.loadSchedule();
          this.loadJadwalEfektif();
          this.loadTugasPortal();
          this.searchMateriQuery = '';
          this.loadMateri();
          this.loadPerubahan();
          this.loadSemester();
        }
      }

      try {
        const newUrl = '/c/' + this.selectedClassSlug;
        if (window.location.pathname !== newUrl) {
          window.history.replaceState({}, '', newUrl);
        }
      } catch (e) { }
    },

    parseClass(code) {
      if (!code) return { prodi: 'D4', smt: '3', abjad: 'A' };
      const str = String(code).trim();
      const m = str.match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-SMT(\d+)-([A-Za-z0-9]+)$/i);
      if (m) {
        return { prodi: m[1].toUpperCase(), smt: m[2], abjad: m[3].toUpperCase() };
      }
      const mYear = str.match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-(\d{4})-([A-Za-z0-9]+)$/i);
      if (mYear) {
        return { prodi: mYear[1].toUpperCase(), smt: '1', abjad: mYear[3].toUpperCase() };
      }
      const mShort = str.match(/^([A-Za-z0-9]+)-[A-Za-z0-9]+-(\d+)([A-Za-z0-9]+)$/i);
      if (mShort) {
        return { prodi: mShort[1].toUpperCase(), smt: mShort[2], abjad: mShort[3].toUpperCase() };
      }
      return { prodi: 'D4', smt: '3', abjad: 'A' };
    },

    get selectedClassFormatted() {
      if (!this.selectedClass) return 'kelas ini';
      const raw = String(this.selectedClass).trim();
      const mYear = raw.match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-(\d{4})-([A-Za-z0-9]+)$/i);
      if (mYear) {
        return `${mYear[1].toUpperCase()}-${mYear[2].toUpperCase()} 1${mYear[4].toUpperCase()}`;
      }
      const mSmt = raw.match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-SMT(\d+)-([A-Za-z0-9]+)$/i);
      if (mSmt) {
        return `${mSmt[1].toUpperCase()}-${mSmt[2].toUpperCase()} ${mSmt[3]}${mSmt[4].toUpperCase()}`;
      }
      const mShort = raw.match(/^([A-Za-z0-9]+)-([A-Za-z0-9]+)-([0-9]+[A-Za-z]+)$/i);
      if (mShort) {
        return `${mShort[1].toUpperCase()}-${mShort[2].toUpperCase()} ${mShort[3].toUpperCase()}`;
      }
      if (/^[A-Za-z0-9]+-[A-Za-z0-9]+\s+[0-9A-Za-z]+$/i.test(raw)) {
        return raw.toUpperCase();
      }
      return raw;
    },

    get selectedClassMeta() {
      if (!this.selectedClass) return { prodi: '', smt: '', abjad: '', label: 'Kelas belum dipilih' };
      const p = this.parseClass(this.selectedClass);
      const prodiName = p.prodi === 'D4' ? 'D4 Teknik Informatika' : (p.prodi === 'D3' ? 'D3 Teknik Informatika' : p.prodi);
      return {
        ...p,
        prodiName,
        label: `${prodiName} · Semester ${p.smt} · Kelas ${p.abjad}`
      };
    },

    toggleCombobox() {
      this.comboboxOpen = !this.comboboxOpen;
      if (this.comboboxOpen) {
        setTimeout(() => {
          const inp = document.getElementById('search-class-input');
          if (inp) inp.focus();
        }, 50);
      }
    },

    get flatClassGroups() {
      const q = this.classQuery.trim().toLowerCase();
      const groups = [
        {
          title: 'D4 Teknik Informatika',
          prodi: 'D4',
          badge: 'Sarjana Terapan',
          items: this.classList.filter(c => this.parseClass(c).prodi === 'D4')
        },
        {
          title: 'D3 Teknik Informatika',
          prodi: 'D3',
          badge: 'Diploma Tiga',
          items: this.classList.filter(c => this.parseClass(c).prodi === 'D3')
        }
      ];

      return groups.map(g => {
        if (this.classFilterProdi !== 'ALL' && g.prodi !== this.classFilterProdi) {
          return { ...g, items: [] };
        }
        const matched = g.items.filter(c => {
          if (!q) return true;
          const p = this.parseClass(c);
          const searchStr = `${c} ${p.prodi} ${p.smt} ${p.abjad} semester ${p.smt} kelas ${p.abjad} smt${p.smt} ${p.smt}${p.abjad}`.toLowerCase();
          return searchStr.includes(q);
        }).map(c => {
          const p = this.parseClass(c);
          return {
            code: c,
            smt: p.smt,
            abjad: p.abjad,
            label: `Semester ${p.smt} · Kelas ${p.abjad}`
          };
        });
        return { ...g, items: matched };
      }).filter(g => g.items.length > 0);
    },

    get totalFilteredCount() {
      return this.flatClassGroups.reduce((acc, g) => acc + g.items.length, 0);
    },

    selectAndCloseClass(c) {
      this.pilihKelas(c);
      this.comboboxOpen = false;
      this.classQuery = '';
    },

    get availableProdis() {
      const list = this.classList.map(c => this.parseClass(c).prodi);
      const unique = Array.from(new Set(list));
      return unique.length ? unique : ['D4', 'D3'];
    },

    get availableSemesters() {
      const list = this.classList
        .map(c => this.parseClass(c))
        .filter(p => p.prodi === this.selectedProdi)
        .map(p => p.smt);
      const unique = Array.from(new Set(list)).sort((a, b) => Number(a) - Number(b));
      return unique.length ? unique : ['1', '3', '5', '7'];
    },

    get availableAbjads() {
      const list = this.classList
        .map(c => this.parseClass(c))
        .filter(p => p.prodi === this.selectedProdi && p.smt === this.selectedSemester)
        .map(p => p.abjad);
      const unique = Array.from(new Set(list)).sort();
      return unique.length ? unique : ['A', 'B', 'C', 'D'];
    },

    setProdi(prodi) {
      this.selectedProdi = prodi;
      const semList = this.availableSemesters;
      if (!semList.includes(this.selectedSemester)) {
        this.selectedSemester = semList[0] || '1';
      }
      const abjadList = this.availableAbjads;
      if (!abjadList.includes(this.selectedAbjad)) {
        this.selectedAbjad = abjadList[0] || 'A';
      }
      this.syncClassFromSegments();
    },

    setSemester(smt) {
      this.selectedSemester = smt;
      const abjadList = this.availableAbjads;
      if (!abjadList.includes(this.selectedAbjad)) {
        this.selectedAbjad = abjadList[0] || 'A';
      }
      this.syncClassFromSegments();
    },

    setAbjad(abjad) {
      this.selectedAbjad = abjad;
      this.syncClassFromSegments();
    },

    syncClassFromSegments() {
      const targetPattern = new RegExp(`^${this.selectedProdi}-.*-SMT${this.selectedSemester}-${this.selectedAbjad}$`, 'i');
      const found = this.classList.find(c => targetPattern.test(c));
      const targetClass = found || `${this.selectedProdi}-TI-SMT${this.selectedSemester}-${this.selectedAbjad}`;
      this.pilihKelas(targetClass);
    },

    syncSegmentsFromClass() {
      if (!this.selectedClass) return;
      const p = this.parseClass(this.selectedClass);
      this.selectedProdi = p.prodi;
      this.selectedSemester = p.smt;
      this.selectedAbjad = p.abjad;
    },

    go(v) {
      this.pageState = null;
      if (!this.knownViews.includes(v)) { this.showPageError('404'); return; }
      this.view = v;
      this.drawer = false;
      if (v === 'jadwal') this.loadJadwalEfektif();
      if (v === 'courses') this.loadCourses();
      if (v === 'arsip') this.loadSemester();
      if (v === 'perubahan') this.loadPerubahan();
      if (v === 'materi') {
        this.searchMateriQuery = '';
        this.loadMateri();
      }
      window.scrollTo({ top: 0 });
    },

    soon(fitur) { this.showToast(`${fitur}: fitur belum tersedia.`); },

    async loadCourses() {
      this.coursesLoading = true; this.coursesError = '';
      try { this.courses = await API.getPortalCourses(this.selectedClassSlug); }
      catch (e) { this.courses = []; this.coursesError = e.message || 'Mata kuliah gagal dimuat.'; }
      finally { this.coursesLoading = false; }
    },

    bukaCourse(course) { this.selectedCourse = course; this.go('course-detail'); },

    async loadClasses() {
      const pathMatch = window.location.pathname.match(/\/c\/([^/]+)/);
      const urlParams = new URLSearchParams(window.location.search);
      const slugFromUrl = (pathMatch && pathMatch[1]) || urlParams.get('c') || urlParams.get('kelas');

      try {
        const data = await API.getClasses();
        if (data && Array.isArray(data.classes)) {
          this.classList = data.classes;
        }
        if (slugFromUrl) {
          const match = (this.classList || []).find(c => c.toLowerCase().replace(/\s+/g, '-') === slugFromUrl.toLowerCase());
          this.selectedClass = match || decodeURIComponent(slugFromUrl);
        } else if (data && data.default_class) {
          this.selectedClass = data.default_class;
        }
      } catch (e) { /* fallback */ }

      this.gateStep = 'pin';

      if (!this.selectedClass) {
        try {
          const st = await API.getStatus();
          if (st && st.default_class) this.selectedClass = st.default_class;
          if (st && Array.isArray(st.classes)) this.classList = st.classes;
        } catch (e) { /* kosong */ }
      }

      if (!this.selectedClass && this.classList.length > 0) {
        this.selectedClass = this.classList[0];
      }
      this.syncSegmentsFromClass();

      const savedClass = localStorage.getItem('portal_class');
      if (savedClass && savedClass === this.selectedClassSlug) {
        this.unlocked = localStorage.getItem('portal_pin_ok') === '1';
      } else if (savedClass && savedClass !== this.selectedClassSlug) {
        this.unlocked = false;
      }
    },

    normalisasiEfektif(items, hariLabel) {
      return (items || []).map((it, i) => {
        let act = String(it.activity_type || '').trim().toUpperCase();
        const matkulTitle = it.offering || it.title || 'Mata Kuliah';
        const roomName = it.room || '';
        // Backend mengirim TEORI/PRAKTIKUM/PRAKTIK/KULIAH, sedangkan template memeriksa
        // PRACTICE/THEORY. Samakan dulu agar badge dan warna kartu konsisten.
        if (act === 'PRAKTIKUM' || act === 'PRAKTIK' || act === 'LAB') {
          act = 'PRACTICE';
        } else if (act === 'TEORI' || act === 'KULIAH') {
          act = 'THEORY';
        }
        if (!act || act === 'THEORY') {
          const mLower = matkulTitle.toLowerCase();
          const rLower = roomName.toLowerCase();
          if (mLower.includes('praktik') || rLower.includes('lab')) {
            act = 'PRACTICE';
          } else {
            act = act || 'THEORY';
          }
        }
        return {
          id: it.id || `ef-${i}`,
          hari: hariLabel || '',
          kind: it.kind || 'REGULER',
          activityType: act,
          matkul: matkulTitle,
          code: it.code || '',
          dosen: Array.isArray(it.lecturers) ? it.lecturers.join(', ') : (it.lecturers || ''),
          ruang: roomName,
          link: it.meeting_link || '',
          timeStart: (it.starts_at || '').slice(0, 5),
          timeEnd: (it.ends_at || '').slice(0, 5),
          pj: it.pj || '',
          pjPhone: it.pj_phone || '',
          originDate: it.origin_occurrence_date || '',
          reason: it.reason || ''
        };
      });
    },

    cleanMatkul(matkul) {
      if (!matkul) return '';
      return String(matkul).replace(/\s*\((?:Teori|Praktikum|Praktik|Pengganti)\)/gi, '').trim();
    },

    cleanRuang(ruang) {
      if (!ruang) return 'Tanpa ruang';
      return String(ruang).replace(/-/g, '-').trim();
    },

    fmtAsalPengganti(isoDate, reason) {
      if (isoDate) {
        try {
          const d = new Date(isoDate + 'T00:00:00+07:00');
          const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
          const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
          return `Pengganti sesi ${days[d.getDay()]}, ${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
        } catch (e) { }
      }
      if (reason) return reason;
      return 'Kuliah pengganti resmi';
    },

    getInisial(name) {
      if (!name) return '—';
      const clean = String(name).replace(/^(bpk\.?|ibu\.?|dr\.?|dra\.?|drs\.?)\s+/i, '').trim();
      const parts = clean.split(/[\s,]+/).filter(Boolean);
      if (parts.length === 0) return '—';
      if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
      return (parts[0][0] + parts[1][0]).toUpperCase();
    },

    get tugasTerkaitDetailSesi() {
      if (!this.detailSesi || !this.detailSesi.matkul) return [];
      const target = this.detailSesi.matkul.toLowerCase().trim();
      return (this.tasks || []).filter(t => {
        if (!t || !t.matkul) return false;
        const tm = t.matkul.toLowerCase().trim();
        return tm === target || target.includes(tm) || tm.includes(target);
      });
    },

    chatWhatsAppPJ(s) {
      if (!s) return;
      const raw = s.pjPhone || '';
      const clean = raw.replace(/[^0-9]/g, '');
      if (clean && clean.length >= 9) {
        const text = encodeURIComponent(`Halo PJ ${s.matkul}, saya mahasiswa ingin menanyakan perihal perkuliahan.`);
        window.open(`https://wa.me/${clean}?text=${text}`, '_blank', 'noopener,noreferrer');
      } else {
        this.showToast(`Kontak PJ ${s.pj ? '(' + s.pj + ')' : ''} dapat dihubungi melalui grup WhatsApp kelas.`);
      }
    },

    salinInfoSesi(s) {
      if (!s) return;
      const text = `📅 *Jadwal Perkuliahan: ${s.matkul}*\n` +
        `🗓️ Hari: ${this.formatTanggalLengkap(s.hari)}\n` +
        `⏰ Waktu: ${s.timeStart} – ${s.timeEnd} WIB\n` +
        `🏢 Ruangan: ${s.ruang ? 'Ruang ' + s.ruang : (s.link ? 'Kuliah Daring Zoom' : '—')}\n` +
        `👨‍🏫 Dosen: ${s.dosen || '—'}\n` +
        `👤 PJ Matkul: ${s.pj || 'Belum ditugaskan'}` +
        (s.link ? `\n🔗 Link Zoom: ${s.link}` : '') +
        (s.kind === 'PENGGANTI' ? `\n⚠️ Keterangan: ${this.fmtAsalPengganti(s.originDate, s.reason)}` : '');
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          this.showToast('Informasi sesi berhasil disalin ke clipboard!');
        }).catch(() => {
          this.showToast('Gagal menyalin informasi sesi.');
        });
      } else {
        this.showToast('Informasi sesi berhasil disalin.');
      }
    },

    async prevWeek() {
      this.weekOffset--;
      await this.loadJadwalPekan();
    },

    async nextWeek() {
      this.weekOffset++;
      await this.loadJadwalPekan();
    },

    async mingguIni() {
      this.weekOffset = 0;
      this.hari = this.todayName;
      await this.loadJadwalPekan();
    },

    lihatJadwalBesok() {
      const days = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      const curIdx = days.indexOf(this.hari);
      const nextIdx = (curIdx + 1) % days.length;
      if (nextIdx === 0) {
        this.weekOffset++;
        this.hari = 'Senin';
        this.loadJadwalPekan();
      } else {
        this.pilihHari(days[nextIdx]);
      }
    },

    get sheetSesiTransform() {
      if (this.sheetSesiDragging && this.sheetSesiOffsetY > 0) {
        return `transform: translateY(${this.sheetSesiOffsetY}px); transition: none;`;
      }
      if (this.sheetSesiClosing) {
        return 'transform: translateY(100%); transition: transform 0.28s cubic-bezier(0.16, 1, 0.3, 1);';
      }
      if (this.sheetSesiResetting) {
        return 'transform: translateY(0px); transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);';
      }
      return '';
    },

    bukaDetailSesi(s) {
      if (!s) return;
      this.detailSesi = s;
      this.sheetSesiOffsetY = 0;
      this.sheetSesiDragging = false;
      this.sheetSesiClosing = false;
      this.sheetSesiResetting = false;
      this.detailModalOpen = true;
    },

    tutupDetailSesi() {
      this.detailModalOpen = false;
      this.sheetSesiDragging = false;
      setTimeout(() => {
        if (!this.detailModalOpen) {
          this.detailSesi = null;
          this.sheetSesiClosing = false;
          this.sheetSesiOffsetY = 0;
        }
      }, 250);
    },

    onSesiPointerDown(e) {
      if (window.innerWidth >= 768) return;
      if (e.target.closest('button, a, input, [role="button"]')) return;
      this.sheetSesiStartY = e.clientY;
      this.sheetSesiDragging = true;
      this.sheetSesiClosing = false;
      this.sheetSesiResetting = false;
      try {
        if (e.target && e.target.setPointerCapture) {
          e.target.setPointerCapture(e.pointerId);
        }
      } catch (_) {}
    },

    onSesiPointerMove(e) {
      if (!this.sheetSesiDragging) return;
      const delta = e.clientY - this.sheetSesiStartY;
      if (delta > 0) {
        this.sheetSesiOffsetY = delta;
      } else {
        this.sheetSesiOffsetY = 0;
      }
    },

    onSesiPointerUp(e) {
      if (!this.sheetSesiDragging) return;
      this.sheetSesiDragging = false;
      try {
        if (e.target && e.target.releasePointerCapture) {
          e.target.releasePointerCapture(e.pointerId);
        }
      } catch (_) {}
      if (this.sheetSesiOffsetY > 75) {
        this.sheetSesiClosing = true;
        this.tutupDetailSesi();
      } else {
        this.sheetSesiResetting = true;
        this.sheetSesiOffsetY = 0;
        setTimeout(() => {
          this.sheetSesiResetting = false;
        }, 220);
      }
    },

    get sheetRiwayatTransform() {
      if (this.sheetRiwayatDragging && this.sheetRiwayatOffsetY > 0) {
        return `transform: translateY(${this.sheetRiwayatOffsetY}px); transition: none;`;
      }
      if (this.sheetRiwayatClosing) {
        return 'transform: translateY(100%); transition: transform 0.28s cubic-bezier(0.16, 1, 0.3, 1);';
      }
      if (this.sheetRiwayatResetting) {
        return 'transform: translateY(0px); transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);';
      }
      return '';
    },

    bukaRiwayatPerubahan() {
      this.sheetRiwayatOffsetY = 0;
      this.sheetRiwayatDragging = false;
      this.sheetRiwayatClosing = false;
      this.sheetRiwayatResetting = false;
      this.modalRiwayatOpen = true;
      if (!this.perubahanList || this.perubahanList.length === 0) {
        this.loadPerubahan();
      }
    },

    tutupRiwayatPerubahan() {
      this.modalRiwayatOpen = false;
      this.sheetRiwayatDragging = false;
      setTimeout(() => {
        if (!this.modalRiwayatOpen) {
          this.sheetRiwayatClosing = false;
          this.sheetRiwayatOffsetY = 0;
        }
      }, 250);
    },

    onRiwayatPointerDown(e) {
      if (window.innerWidth >= 768) return;
      if (e.target.closest('button, a, input, [role="button"]')) return;
      this.sheetRiwayatStartY = e.clientY;
      this.sheetRiwayatDragging = true;
      this.sheetRiwayatClosing = false;
      this.sheetRiwayatResetting = false;
      try {
        if (e.target && e.target.setPointerCapture) {
          e.target.setPointerCapture(e.pointerId);
        }
      } catch (_) {}
    },

    onRiwayatPointerMove(e) {
      if (!this.sheetRiwayatDragging) return;
      const delta = e.clientY - this.sheetRiwayatStartY;
      if (delta > 0) {
        this.sheetRiwayatOffsetY = delta;
      } else {
        this.sheetRiwayatOffsetY = 0;
      }
    },

    onRiwayatPointerUp(e) {
      if (!this.sheetRiwayatDragging) return;
      this.sheetRiwayatDragging = false;
      try {
        if (e.target && e.target.releasePointerCapture) {
          e.target.releasePointerCapture(e.pointerId);
        }
      } catch (_) {}
      if (this.sheetRiwayatOffsetY > 75) {
        this.sheetRiwayatClosing = true;
        this.tutupRiwayatPerubahan();
      } else {
        this.sheetRiwayatResetting = true;
        this.sheetRiwayatOffsetY = 0;
        setTimeout(() => {
          this.sheetRiwayatResetting = false;
        }, 220);
      }
    },

    salinRingkasanSesi(s) {
      if (!s) return;
      const tgl = this.tanggalPekanFormatted[s.hari] || s.hari;
      const teks = `[Jadwal Kuliah - ${s.matkul}]\n` +
        `📅 Hari/Tgl: ${s.hari}, ${tgl}\n` +
        `⏰ Waktu: ${s.timeStart} – ${s.timeEnd} WIB\n` +
        `📍 Tempat: ${s.ruang ? 'Ruang ' + s.ruang : (s.link ? 'Kuliah Daring' : '—')}\n` +
        `👨‍🏫 Dosen: ${s.dosen || '—'}\n` +
        (s.pj ? `👤 PJ: ${s.pj}\n` : '') +
        (s.kind === 'PENGGANTI' ? `ℹ️ Keterangan: ${this.fmtAsalPengganti(s.originDate, s.reason)}\n` : '') +
        (s.link ? `🔗 Link: ${s.link}\n` : '');
      if (navigator && navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(teks).then(() => {
          this.showToast('Ringkasan sesi berhasil disalin ke papan klip!');
        }).catch(() => {
          this.showToast('Ringkasan sesi disalin.');
        });
      } else {
        this.showToast('Ringkasan sesi disalin.');
      }
    },

    hubungiPJ(s) {
      this.chatWhatsAppPJ(s);
    },

    getTugasTerkait(matkul) {
      if (!matkul || !Array.isArray(this.tasks)) return [];
      const m = String(matkul).toLowerCase().trim();
      return this.tasks.filter(t => {
        const tm = String(t.matkul || '').toLowerCase().trim();
        return tm && (tm.includes(m) || m.includes(tm));
      }).slice(0, 2);
    },

    getStatusWaktu(s) {
      if (!s || !s.timeStart) return '';
      try {
        const timeParts = new Intl.DateTimeFormat('en-GB', {
          timeZone: 'Asia/Jakarta',
          hour: 'numeric',
          minute: 'numeric',
          hour12: false
        }).formatToParts(new Date());
        const h = parseInt(timeParts.find(p => p.type === 'hour')?.value || '0', 10);
        const m = parseInt(timeParts.find(p => p.type === 'minute')?.value || '0', 10);
        const cur = h * 60 + m;
        const [h1, m1] = String(s.timeStart || '').replace('.', ':').split(':').map(Number);
        const [h2, m2] = String(s.timeEnd || '').replace('.', ':').split(':').map(Number);
        const start = h1 * 60 + m1;
        const end = h2 * 60 + m2;
        if (cur >= start && cur <= end) return 'Sedang berlangsung';
        if (cur < start) return 'Akan datang';
        return 'Selesai';
      } catch (e) {
        return '';
      }
    },

    getInitials(name) {
      if (!name) return 'DS';
      const cleaned = String(name)
        .replace(/^(Bpk\.|Ibu\.|Pak\.|Dr\.|Prof\.|Ir\.)\s*/i, '')
        .replace(/,\s*[A-Za-z.\s]+$/i, '')
        .replace(/\(.*\)/g, '')
        .trim();
      const parts = cleaned.split(/\s+/).filter(Boolean);
      if (parts.length >= 2) {
        return (parts[0][0] + parts[1][0]).toUpperCase();
      }
      if (parts.length === 1 && parts[0].length >= 2) {
        return parts[0].slice(0, 2).toUpperCase();
      }
      return (cleaned[0] || 'D').toUpperCase();
    },

    getKodeMatkul(s) {
      if (!s) return 'TI2104';
      if (s.code || s.kode) return s.code || s.kode;
      const m = String(s.matkul || '').toLowerCase();
      if (m.includes('aljabar')) return 'TI2104';
      if (m.includes('web')) return 'TI2105';
      if (m.includes('basis data')) return 'TI2103';
      if (m.includes('objek') || m.includes('pbo')) return 'TI2102';
      return 'TI2101';
    },

    formatWaktuPublikasi(iso) {
      if (!iso) return 'Terbit oleh KM';
      try {
        const d = new Date(iso);
        if (isNaN(d)) return String(iso);
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        const day = d.getDate();
        const month = months[d.getMonth()];
        const hh = String(d.getHours()).padStart(2, '0');
        const mm = String(d.getMinutes()).padStart(2, '0');
        return `${day} ${month}, ${hh}:${mm} WIB oleh KM`;
      } catch (e) {
        return 'Terbit oleh KM';
      }
    },

    formatWaktuEfektif(ch) {
      if (!ch) return '';
      try {
        const start = ch.starts_at ? new Date(ch.starts_at) : null;
        const end = ch.ends_at ? new Date(ch.ends_at) : null;
        const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
        if (start && !isNaN(start)) {
          const dayName = days[start.getDay()];
          const d = start.getDate();
          const m = months[start.getMonth()];
          const y = start.getFullYear();
          const h1 = String(start.getHours()).padStart(2, '0');
          const m1 = String(start.getMinutes()).padStart(2, '0');
          let jamStr = `${h1}:${m1}`;
          if (end && !isNaN(end)) {
            const h2 = String(end.getHours()).padStart(2, '0');
            const m2 = String(end.getMinutes()).padStart(2, '0');
            jamStr += ` - ${h2}:${m2}`;
          }
          jamStr += ' WIB';
          const ruangStr = ch.room ? ` • Ruang ${ch.room}` : '';
          return `${dayName}, ${d} ${m} ${y} • ${jamStr}${ruangStr}`;
        }
      } catch (e) { }
      return ch.room ? `Ruang ${ch.room}` : '';
    },

    async loadSchedule() {
      try {
        const raw = await API.fetchLegacySchedule(this.selectedClass);
        this.fullSchedule = (raw || []).map((s, i) => {
          const parts = String(s.jam || '').split('-').map(x => x.trim().replace('.', ':'));
          return {
            id: `sch-${i}`, hari: s.hari, jam: s.jam, matkul: s.matkul,
            dosen: s.dosen, ruang: s.ruang,
            timeStart: parts[0] || '', timeEnd: parts[1] || ''
          };
        });
      } catch (e) { this.fullSchedule = []; }
    },

    // Jadwal efektif (pola + perubahan terbit) untuk seluruh pekan.
    async loadJadwalPekan() {
      this.jadwalLoading = true;
      this.jadwalError = '';
      const slug = this.selectedClassSlug;
      const days = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
      try {
        const results = await Promise.all(days.map(d => {
          const tgl = this.tanggalPekan[d];
          return API.getPortalSchedule(slug, tgl);
        }));

        let totalV1Items = results.reduce((sum, r) => sum + ((r && Array.isArray(r.items)) ? r.items.length : 0), 0);
        if (totalV1Items === 0 && (!this.fullSchedule || this.fullSchedule.length === 0)) {
          await this.loadSchedule();
        }
        const useLegacyFallback = totalV1Items === 0 && (this.fullSchedule || []).length > 0;

        const newPekan = {};
        days.forEach((d, idx) => {
          const res = results[idx];
          if (!useLegacyFallback && res && Array.isArray(res.items)) {
            newPekan[d] = this.normalisasiEfektif(res.items, d);
          } else {
            newPekan[d] = (this.fullSchedule || []).filter(s => s.hari === d).map(s => {
              const mLower = String(s.matkul || '').toLowerCase();
              const rLower = String(s.ruang || '').toLowerCase();
              const isPrak = mLower.includes('praktik') || rLower.includes('lab');
              return {
                id: s.id, hari: s.hari, kind: 'REGULER', activityType: isPrak ? 'PRACTICE' : 'THEORY',
                matkul: s.matkul, dosen: s.dosen || '', ruang: s.ruang || '',
                timeStart: s.timeStart || '', timeEnd: s.timeEnd || '',
                link: '', pj: '', originDate: '', reason: ''
              };
            });
          }
        });
        this.jadwalPekan = newPekan;
        this.jadwalEfektif = this.jadwalPekan[this.hari] || [];
        if (this.weekOffset === 0) {
          this.jadwalCacheHariIni = this.jadwalPekan[this.todayName] || [];
        }
      } catch (e) {
        this.jadwalError = 'Jadwal belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.jadwalLoading = false;
      }
    },

    async loadJadwalEfektif() {
      await this.loadJadwalPekan();
    },

    pilihHari(d) {
      this.hari = d;
      this.jadwalEfektif = this.jadwalPekan[d] || [];
    },

    async loadTugasPortal() {
      this.tugasLoading = true; this.tugasError = '';
      const slug = this.selectedClassSlug;
      try {
        const grup = ['hari_ini', 'minggu_ini', 'mendatang', 'terlewat'];
        const hasil = await Promise.all(grup.map(g =>
          API.getPortalTasks(slug, g)
            .then(data => ({ ok: true, data: data }))
            .catch(err => ({ ok: false, err: err }))
        ));
        if (hasil.every(h => !h.ok)) {
          const st = hasil[0].err && hasil[0].err.status;
          this.tugasGrup = { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] };
          this.tasks = [];
          this.tugasError = st === 404
            ? 'Kelas tidak ditemukan di portal. Periksa kode kelas atau hubungi KM.'
            : st === 401
              ? 'Kelas ini dilindungi kode akses. Kunci portal lalu masukkan kode dari grup kelas.'
              : 'Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
          return;
        }
        const petakan = (arr) => (arr || []).map(t => {
          const deadline = t.deadline_at;
          const u = this.deadlineBadge(deadline, t.completed_at || t.is_completed);
          return {
            id: t.id, matkul: t.matkul || t.offering || 'Mata Kuliah',
            title: t.title || '', deskripsi: t.deskripsi || t.instructions || t.title || '',
            instructions: t.instructions || '',
            task_type: t.task_type || '',
            submission_text: t.submission_text || '',
            submission_url: t.submission_url || '',
            completed_at: t.completed_at || null,
            is_completed: !!t.is_completed,
            review_state: t.review_state || 'NOT_REVIEWED',
            deadline: this.fmtDeadlineID(deadline), deadline_at: deadline,
            urgency: u.level, countdown: u.badge
          };
        });
        this.tugasGrup = {
          hari_ini: petakan(hasil[0].ok ? hasil[0].data : []),
          minggu_ini: petakan(hasil[1].ok ? hasil[1].data : []),
          mendatang: petakan(hasil[2].ok ? hasil[2].data : []),
          terlewat: petakan(hasil[3].ok ? hasil[3].data : [])
        };
        const gabung = new Map();
        [...this.tugasGrup.mendatang, ...this.tugasGrup.minggu_ini, ...this.tugasGrup.hari_ini, ...this.tugasGrup.terlewat]
          .forEach(t => { if (!gabung.has(String(t.id))) gabung.set(String(t.id), t); });
        this.tasks = Array.from(gabung.values());
      } catch (e) {
        this.tugasGrup = { hari_ini: [], minggu_ini: [], mendatang: [], terlewat: [] };
        this.tasks = [];
        this.tugasError = 'Tugas belum dapat dimuat. Periksa koneksi lalu coba lagi.';
      } finally {
        this.tugasLoading = false;
      }
    },

    async loadTasks() { await this.loadTugasPortal(); },

    async bukaDetailTugas(t) {
      this.detailTugas = { memuat: true };
      this.detailMateri = [];
      this.view = 'detail-tugas';
      window.scrollTo({ top: 0 });
      try {
        const d = await API.getPortalTaskDetail(this.selectedClassSlug, t.id);
        const info = (d && (d.task || d)) || null;
        if (!info) throw { code: 'NOT_FOUND' };
        const u = this.deadlineBadge(info.deadline_at, info.completed_at || info.is_completed);
        this.detailTugas = {
          id: info.id, matkul: info.offering || '', title: info.title || '',
          instructions: info.instructions || '',
          deadline: this.fmtDeadlineID(info.deadline_at),
          deadline_at: info.deadline_at,
          task_type: info.task_type || '',
          submission_text: info.submission_text || '', submission_url: info.submission_url || '',
          version: info.version || '',
          completed_at: info.completed_at || null,
          is_completed: !!info.is_completed,
          review_state: info.review_state || 'NOT_REVIEWED',
          urgency: u.level, countdown: u.badge
        };
        this.detailMateri = (d && d.materials) || [];
      } catch (e) {
        if (e && e.code === 'NOT_FOUND') { this.showPageError('404'); return; }
        if (e && e.code === 'LOAD_FAILED') { this.showPageError('500'); return; }
        this.detailTugas = { hilang: true };
        this.detailMateri = [];
      }
    },

    async loadMateri() {
      this.materiLoading = true;
      this.materiError = '';
      try {
        const items = await API.getPortalMaterials(this.selectedClassSlug);
        this.materiList = Array.isArray(items) ? items : [];
      } catch (e) {
        this.materiList = [];
        this.materiError = 'Materi gagal dimuat. Coba lagi.';
      } finally {
        this.materiLoading = false;
      }
    },

    get materiCategories() {
      const all = this.materiList || [];
      const totalCount = all.length;

      const umumCount = all.filter(m => {
        const c = (m.course_name || '').toLowerCase();
        return !m.course_name || c.includes('umum') || c.includes('silabus') || c.includes('rps');
      }).length;

      const list = [
        { key: 'SEMUA', label: 'Semua Materi', shortLabel: 'Semua', count: totalCount },
        { key: 'UMUM', label: 'Umum Kelas (Silabus & RPS)', shortLabel: 'Umum Kelas', count: umumCount },
      ];

      const foundKeys = new Set();
      const dynamicCourses = [];

      (this.courses || []).forEach(c => {
        const name = (c.course_name || c.name || '').trim();
        if (name) dynamicCourses.push(name);
      });
      Object.values(this.jadwalPekan || {}).flat().forEach(s => {
        const name = this.cleanMatkul(s.matkul || '');
        if (name) dynamicCourses.push(name);
      });
      all.forEach(m => {
        const name = (m.course_name || '').trim();
        if (name) dynamicCourses.push(name);
      });

      dynamicCourses.forEach(cName => {
        const lower = cName.toLowerCase();
        if (cName && !foundKeys.has(lower) && !lower.includes('umum') && !lower.includes('silabus') && !lower.includes('rps')) {
          foundKeys.add(lower);
          const count = all.filter(item => (item.course_name || '').trim().toLowerCase() === lower).length;
          list.push({
            key: cName,
            label: cName,
            shortLabel: cName.length > 14 ? cName.split(' ').slice(0, 2).join(' ') : cName,
            count,
          });
        }
      });

      return list;
    },

    get filteredMateriList() {
      let list = [...(this.materiList || [])];
      const filter = this.filterMateriMatkul;

      if (filter === 'UMUM') {
        list = list.filter(m => {
          const c = (m.course_name || '').toLowerCase();
          return !m.course_name || c.includes('umum') || c.includes('silabus') || c.includes('rps');
        });
      } else if (filter && filter !== 'SEMUA') {
        list = list.filter(m => (m.course_name || '').trim().toLowerCase() === filter.toLowerCase());
      }

      const q = (this.searchMateriQuery || '').trim().toLowerCase();
      if (q) {
        list = list.filter(m => {
          const title = (m.title || '').toLowerCase();
          const course = (m.course_name || '').toLowerCase();
          const desc = (m.description || '').toLowerCase();
          return title.includes(q) || course.includes(q) || desc.includes(q);
        });
      }

      if (this.filterMateriSesi === 'TEORI') {
        list = list.filter(m => this.getMateriActivity(m) === 'Teori');
      } else if (this.filterMateriSesi === 'PRAKTIKUM') {
        list = list.filter(m => this.getMateriActivity(m) === 'Praktikum');
      }

      if (this.sortMateri === 'terlama') {
        list.sort((a, b) => (a.id || 0) - (b.id || 0));
      } else {
        list.sort((a, b) => (b.id || 0) - (a.id || 0));
      }

      return list;
    },

    get materiSesiCounts() {
      const all = this.materiList || [];
      const filter = this.filterMateriMatkul;
      let scoped = all;
      if (filter === 'UMUM') {
        scoped = scoped.filter(m => {
          const c = (m.course_name || '').toLowerCase();
          return !m.course_name || c.includes('umum') || c.includes('silabus') || c.includes('rps');
        });
      } else if (filter && filter !== 'SEMUA') {
        scoped = scoped.filter(m => (m.course_name || '').trim().toLowerCase() === filter.toLowerCase());
      }
      const q = (this.searchMateriQuery || '').trim().toLowerCase();
      if (q) {
        scoped = scoped.filter(m => {
          const title = (m.title || '').toLowerCase();
          const course = (m.course_name || '').toLowerCase();
          const desc = (m.description || '').toLowerCase();
          return title.includes(q) || course.includes(q) || desc.includes(q);
        });
      }

      const total = scoped.length;
      const teori = scoped.filter(m => this.getMateriActivity(m) === 'Teori').length;
      const praktikum = scoped.filter(m => this.getMateriActivity(m) === 'Praktikum').length;
      return { total, teori, praktikum };
    },

    get currentMateriHeading() {
      const count = this.filteredMateriList.length;
      const q = (this.searchMateriQuery || '').trim();
      if (q) {
        return `Hasil Pencarian "${q}" (${count} Berkas)`;
      }
      const sesiSuffix = this.filterMateriSesi === 'PRAKTIKUM' ? ' · Praktikum' : (this.filterMateriSesi === 'TEORI' ? ' · Teori' : '');
      if (this.filterMateriMatkul === 'SEMUA') {
        return `Semua Berkas Materi${sesiSuffix} (${count} Berkas)`;
      }
      if (this.filterMateriMatkul === 'UMUM') {
        return `Umum Kelas (Silabus & RPS) (${count} Berkas)`;
      }
      return `${this.filterMateriMatkul}${sesiSuffix} (${count} Berkas)`;
    },

    getMateriActivity(m) {
      if (!m) return 'Teori';
      const cName = (m.course_name || '').toLowerCase();
      if (cName.includes('umum') || cName.includes('silabus') || cName.includes('rps')) {
        return 'Umum';
      }
      const act = String(m.activity_type || '').toUpperCase();
      if (act === 'PRACTICE' || act === 'PRAKTIKUM' || act === 'LAB') return 'Praktikum';
      if (act === 'THEORY' || act === 'TEORI') return 'Teori';
      if (act === 'GENERAL' || act === 'UMUM') return 'Umum';

      // Fallback deteksi cerdas berdasarkan judul dan deskripsi
      const text = `${m.title || ''} ${m.description || ''}`.toLowerCase();
      if (
        text.includes('praktikum') ||
        text.includes('modul lab') ||
        text.includes('sesi lab') ||
        text.includes('jobsheet') ||
        text.includes('starter') ||
        text.includes('assembly') ||
        text.includes('lab komputer')
      ) {
        return 'Praktikum';
      }
      return 'Teori';
    },

    getMateriActivityTheme(m) {
      const act = this.getMateriActivity(m);
      if (act === 'Praktikum') {
        return 'bg-purple-50 text-purple-700 border border-purple-200';
      }
      if (act === 'Umum') {
        return 'bg-slate-100 text-slate-600 border border-slate-200';
      }
      return 'bg-blue-50 text-blue-700 border border-blue-200';
    },

    getMateriIconTheme(m) {
      const type = this.getMateriType(m);
      if (type === 'Video') {
        return {
          bg: 'bg-purple-50 text-purple-600 border-purple-200',
          badge: 'bg-purple-50 text-purple-700',
        };
      }
      if (type === 'Repositori') {
        return {
          bg: 'bg-emerald-50 text-emerald-600 border-emerald-200',
          badge: 'bg-emerald-50 text-emerald-700',
        };
      }
      if (type === 'Dokumen') {
        return {
          bg: 'bg-rose-50 text-rose-600 border-rose-200',
          badge: 'bg-rose-50 text-rose-700',
        };
      }
      return {
        bg: 'bg-blue-50 text-blue-600 border-blue-200',
        badge: 'bg-blue-50 text-blue-700',
      };
    },

    getMateriType(m) {
      if (!m) return 'Slide';
      const t = String(m.material_type || '').toUpperCase();
      const title = String(m.title || '').toLowerCase();
      if (t === 'MEETING' || title.includes('rekaman') || title.includes('video')) return 'Video';
      if (t === 'REPOSITORY' || title.includes('repositori') || title.includes('starter') || title.includes('template')) return 'Repositori';
      if (t === 'DOCUMENT') {
        if (title.includes('slide')) return 'Slide';
        return 'Dokumen';
      }
      if (title.includes('slide')) return 'Slide';
      return 'Slide';
    },

    getMateriSubtitle(m) {
      if (!m) return '';
      if (m.subtitle_display) return m.subtitle_display;
      const course = m.course_name || 'Umum Kelas';
      const type = this.getMateriType(m);
      if (type === 'Video') return `${course} · Tautan Google Drive`;
      if (type === 'Repositori') return `${course} · GitHub`;

      let format = 'PDF';
      if (m.url && !m.url.includes('.pdf') && !m.url.includes('drive')) format = 'Dokumen';
      const uploader = m.uploader_name ? `Diunggah oleh ${m.uploader_name}` : 'Pengunggah tidak tercatat';
      return `${course} · ${format} · ${uploader}`;
    },

    getMateriDate(m) {
      if (!m) return '-';
      if (m.date_display) return m.date_display;
      if (m.created_at) {
        try {
          const d = new Date(m.created_at);
          if (!isNaN(d.getTime())) {
            const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
            const pad = (n) => String(n).padStart(2, '0');
            return `${pad(d.getDate())} ${months[d.getMonth()]}`;
          }
        } catch (e) { }
      }
      return '-';
    },

    getMateriModalMeta(m) {
      if (!m) return '';
      const type = this.getMateriType(m);
      const act = this.getMateriActivity(m);
      let format = 'Dokumen PDF';
      if (type === 'Video') format = 'Tautan Video (Google Drive)';
      else if (type === 'Repositori') format = 'Repositori Kode (GitHub)';

      const uploader = m.uploader_name ? `Diunggah oleh ${m.uploader_name}` : 'Pengunggah tidak tercatat';
      return `Format: ${format} · Sesi: ${act} · ${uploader}`;
    },

    cleanUrlDisplay(url) {
      if (!url) return '-';
      let clean = url.replace(/^https?:\/\//i, '');
      if (clean.length > 42) {
        clean = clean.substring(0, 39) + '...';
      }
      return clean;
    },

    get sheetMateriTransform() {
      if (this.sheetMateriDragging && this.sheetMateriOffsetY > 0) {
        return `transform: translateY(${this.sheetMateriOffsetY}px); transition: none;`;
      }
      if (this.sheetMateriClosing) {
        return 'transform: translateY(100%); transition: transform 0.28s cubic-bezier(0.16, 1, 0.3, 1);';
      }
      if (this.sheetMateriResetting) {
        return 'transform: translateY(0px); transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);';
      }
      return '';
    },

    bukaDetailMateri(m) {
      this.selectedMateri = m;
      this.sheetMateriOffsetY = 0;
      this.sheetMateriDragging = false;
      this.sheetMateriClosing = false;
      this.sheetMateriResetting = false;
      this.detailMateriModalOpen = true;
    },

    tutupDetailMateri() {
      this.detailMateriModalOpen = false;
      this.sheetMateriDragging = false;
      setTimeout(() => {
        if (!this.detailMateriModalOpen) {
          this.selectedMateri = null;
          this.sheetMateriClosing = false;
          this.sheetMateriOffsetY = 0;
        }
      }, 250);
    },

    onMateriPointerDown(e) {
      if (window.innerWidth >= 768) return;
      if (e.target.closest('button, a, input, [role="button"]')) return;
      this.sheetMateriStartY = e.clientY;
      this.sheetMateriDragging = true;
      this.sheetMateriClosing = false;
      this.sheetMateriResetting = false;
      try {
        if (e.target && e.target.setPointerCapture) {
          e.target.setPointerCapture(e.pointerId);
        }
      } catch (_) {}
    },

    onMateriPointerMove(e) {
      if (!this.sheetMateriDragging) return;
      const delta = e.clientY - this.sheetMateriStartY;
      if (delta > 0) {
        this.sheetMateriOffsetY = delta;
      } else {
        this.sheetMateriOffsetY = 0;
      }
    },

    onMateriPointerUp(e) {
      if (!this.sheetMateriDragging) return;
      this.sheetMateriDragging = false;
      try {
        if (e.target && e.target.releasePointerCapture) {
          e.target.releasePointerCapture(e.pointerId);
        }
      } catch (_) {}
      if (this.sheetMateriOffsetY > 75) {
        this.sheetMateriClosing = true;
        this.tutupDetailMateri();
      } else {
        this.sheetMateriResetting = true;
        this.sheetMateriOffsetY = 0;
        setTimeout(() => {
          this.sheetMateriResetting = false;
        }, 220);
      }
    },

    salinTautanMateri(url) {
      if (!url) {
        this.showToast('Tautan belum tersedia.');
        return;
      }
      if (navigator && navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(() => {
          this.showToast('Tautan berkas berhasil disalin!');
        }).catch(() => {
          this.showToast('Tautan berkas berhasil disalin!');
        });
      } else {
        this.showToast('Tautan berkas berhasil disalin!');
      }
    },

    bukaTautanLangsung(url) {
      if (!url) {
        this.showToast('Tautan berkas belum tersedia.');
        return;
      }
      window.open(url, '_blank', 'noopener,noreferrer');
    },

    get materiGrup() {
      const grup = {};
      (this.materiList || []).forEach(m => {
        const kunci = m.material_type || 'Lainnya';
        if (!grup[kunci]) grup[kunci] = [];
        grup[kunci].push(m);
      });
      return Object.keys(grup).sort().map(k => ({ jenis: k, items: grup[k] }));
    },

    async loadPerubahan() {
      this.perubahanLoading = true;
      try {
        this.perubahanList = await API.getChanges(this.selectedClassSlug).catch(() => null) || [];
      } catch (e) {
        this.perubahanList = [];
      } finally {
        this.perubahanLoading = false;
      }
    },

    async loadSemester() {
      this.semesterLoading = true;
      try {
        this.semesterList = await API.getPortalSemesters(this.selectedClassSlug).catch(() => []);
      } catch (e) {
        this.semesterList = [];
      } finally {
        this.semesterLoading = false;
      }
    },

    labelStatusSemester(st) {
      const s = String(st || '').toUpperCase();
      if (s === 'ACTIVE') return 'Aktif';
      if (s === 'DRAFT') return 'Draf';
      if (s === 'ARCHIVED') return 'Arsip';
      return st || '-';
    },

    labelJenisUbah(kind) {
      const k = String(kind || '').toUpperCase();
      if (k === 'REPLACEMENT') return 'Kelas Pengganti';
      if (k === 'EXTRA') return 'Kelas Tambahan';
      if (k === 'HOLIDAY') return 'Hari Libur';
      if (k === 'SESSION_CANCELLED') return 'Sesi Dibatalkan';
      return kind || '-';
    },

    urgencyOf(label) {
      try {
        const s = String(label || '');
        const hm = s.match(/(\d{1,2})[:.](\d{2})/);
        if (!hm) return { level: 'aman', badge: 'Aktif' };
        const now = new Date(new Date().toLocaleString('en-US', { timeZone: 'Asia/Jakarta' }));
        const target = new Date(now);
        target.setHours(parseInt(hm[1], 10), parseInt(hm[2], 10), 0, 0);
        if (/besok/i.test(s)) target.setDate(target.getDate() + 1);
        else if (!/hari ini/i.test(s)) {
          const dm = s.match(/(\d{1,2})\s+(Jan|Feb|Mar|Apr|Mei|Jun|Jul|Agu|Sep|Okt|Nov|Des)/i);
          if (dm) {
            const months = { jan: 0, feb: 1, mar: 2, apr: 3, mei: 4, jun: 5, jul: 6, agu: 7, sep: 8, okt: 9, nov: 10, des: 11 };
            target.setMonth(months[dm[2].slice(0, 3).toLowerCase()]);
            target.setDate(parseInt(dm[1], 10));
            if (target < new Date(now.getTime() - 86400000)) target.setFullYear(target.getFullYear() + 1);
          }
        }
        const diffH = (target - now) / 3600000;
        if (diffH < 0) return { level: 'mendesak', badge: 'Terlewat' };
        if (diffH < 24) return { level: 'mendesak', badge: 'Besok' };
        if (diffH <= 72) return { level: 'mendekati', badge: `H-${Math.ceil(diffH / 24)}` };
        return { level: 'aman', badge: 'Aktif' };
      } catch (e) {
        return { level: 'aman', badge: 'Aktif' };
      }
    },

    showToast(msg) {
      if (this.toast.timer) clearTimeout(this.toast.timer);
      this.toast.message = msg;
      this.toast.show = true;
      this.toast.timer = setTimeout(() => { this.toast.show = false; }, 3000);
    }
  };
}
