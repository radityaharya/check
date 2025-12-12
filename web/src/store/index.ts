import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { useShallow } from 'zustand/shallow';
import type { Check, Group, Tag, TimeRange } from '@/types';

interface UIState {
  // Theme
  darkMode: boolean;
  setDarkMode: (dark: boolean) => void;
  toggleDarkMode: () => void;

  // Time range
  timeRange: TimeRange;
  setTimeRange: (range: TimeRange) => void;

  // Selected check - store ID and derive check from query cache
  selectedCheckId: number | null;
  setSelectedCheckId: (id: number | null) => void;
  selectCheck: (check: Check | null) => void;

  // Expanded groups - use array internally, expose Set-like API
  expandedGroups: string[];
  toggleGroup: (groupId: string) => void;
  isGroupExpanded: (groupId: string) => boolean;

  // SSE connection status
  sseConnected: boolean;
  setSSEConnected: (connected: boolean) => void;

  // Monitor modal
  monitorModalOpen: boolean;
  editingCheck: Check | null;
  openMonitorModal: (check?: Check | null) => void;
  closeMonitorModal: () => void;

  // Group modal
  groupModalOpen: boolean;
  editingGroup: Group | null;
  openGroupModal: (group?: Group | null) => void;
  closeGroupModal: () => void;

  // Tag modal
  tagModalOpen: boolean;
  editingTag: Tag | null;
  openTagModal: (tag?: Tag | null) => void;
  closeTagModal: () => void;

  // Settings modal
  settingsModalOpen: boolean;
  openSettingsModal: () => void;
  closeSettingsModal: () => void;

  // History modal
  historyModalOpen: boolean;
  historyCheck: Check | null;
  openHistoryModal: (check: Check) => void;
  closeHistoryModal: () => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      // Theme
      darkMode: true,
      setDarkMode: (dark) => {
        set({ darkMode: dark });
        document.documentElement.classList.toggle('dark', dark);
      },
      toggleDarkMode: () => {
        const newValue = !get().darkMode;
        set({ darkMode: newValue });
        document.documentElement.classList.toggle('dark', newValue);
      },

      // Time range
      timeRange: '1d',
      setTimeRange: (range) => set({ timeRange: range }),

      // Selected check
      selectedCheckId: null,
      setSelectedCheckId: (id) => set({ selectedCheckId: id }),
      selectCheck: (check) => set({ selectedCheckId: check?.id ?? null }),

      // Expanded groups - stored as array
      expandedGroups: ['ungrouped'],
      toggleGroup: (groupId) => {
        const { expandedGroups } = get();
        const index = expandedGroups.indexOf(groupId);
        if (index >= 0) {
          set({ expandedGroups: expandedGroups.filter((id) => id !== groupId) });
        } else {
          set({ expandedGroups: [...expandedGroups, groupId] });
        }
      },
      isGroupExpanded: (groupId) => get().expandedGroups.includes(groupId),

      // SSE
      sseConnected: false,
      setSSEConnected: (connected) => set({ sseConnected: connected }),

      // Monitor modal
      monitorModalOpen: false,
      editingCheck: null,
      openMonitorModal: (check = null) =>
        set({ monitorModalOpen: true, editingCheck: check }),
      closeMonitorModal: () =>
        set({ monitorModalOpen: false, editingCheck: null }),

      // Group modal
      groupModalOpen: false,
      editingGroup: null,
      openGroupModal: (group = null) =>
        set({ groupModalOpen: true, editingGroup: group }),
      closeGroupModal: () =>
        set({ groupModalOpen: false, editingGroup: null }),

      // Tag modal
      tagModalOpen: false,
      editingTag: null,
      openTagModal: (tag = null) =>
        set({ tagModalOpen: true, editingTag: tag }),
      closeTagModal: () =>
        set({ tagModalOpen: false, editingTag: null }),

      // Settings modal
      settingsModalOpen: false,
      openSettingsModal: () => set({ settingsModalOpen: true }),
      closeSettingsModal: () => set({ settingsModalOpen: false }),

      // History modal
      historyModalOpen: false,
      historyCheck: null,
      openHistoryModal: (check) =>
        set({ historyModalOpen: true, historyCheck: check }),
      closeHistoryModal: () =>
        set({ historyModalOpen: false, historyCheck: null }),
    }),
    {
      name: 'gocheck-ui-store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        darkMode: state.darkMode,
        timeRange: state.timeRange,
        expandedGroups: state.expandedGroups,
      }),
    }
  )
);

// Selector hooks for better performance (only subscribe to what you need)
export const useTimeRange = () => useUIStore((s) => s.timeRange);
export const useSetTimeRange = () => useUIStore((s) => s.setTimeRange);
export const useDarkMode = () => useUIStore((s) => s.darkMode);
export const useToggleDarkMode = () => useUIStore((s) => s.toggleDarkMode);
export const useSelectedCheckId = () => useUIStore((s) => s.selectedCheckId);
export const useSelectCheck = () => useUIStore((s) => s.selectCheck);
export const useExpandedGroups = () => useUIStore((s) => s.expandedGroups);
export const useToggleGroup = () => useUIStore((s) => s.toggleGroup);
export const useSSEConnected = () => useUIStore((s) => s.sseConnected);
export const useSetSSEConnected = () => useUIStore((s) => s.setSSEConnected);

// Modal hooks - use useShallow to prevent infinite loops from object references
export const useMonitorModal = () =>
  useUIStore(
    useShallow((s) => ({
      isOpen: s.monitorModalOpen,
      editingCheck: s.editingCheck,
      open: s.openMonitorModal,
      close: s.closeMonitorModal,
    }))
  );

export const useGroupModal = () =>
  useUIStore(
    useShallow((s) => ({
      isOpen: s.groupModalOpen,
      editingGroup: s.editingGroup,
      open: s.openGroupModal,
      close: s.closeGroupModal,
    }))
  );

export const useTagModal = () =>
  useUIStore(
    useShallow((s) => ({
      isOpen: s.tagModalOpen,
      editingTag: s.editingTag,
      open: s.openTagModal,
      close: s.closeTagModal,
    }))
  );

export const useSettingsModal = () =>
  useUIStore(
    useShallow((s) => ({
      isOpen: s.settingsModalOpen,
      open: s.openSettingsModal,
      close: s.closeSettingsModal,
    }))
  );

export const useHistoryModal = () =>
  useUIStore(
    useShallow((s) => ({
      isOpen: s.historyModalOpen,
      check: s.historyCheck,
      open: s.openHistoryModal,
      close: s.closeHistoryModal,
    }))
  );
