import { useState, useEffect } from 'react';
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import type { Group } from '@/types';

interface GroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: { name: string; sort_order: number }) => Promise<void>;
  onDelete?: () => Promise<void>;
  editingGroup?: Group | null;
}

export function GroupModal({
  isOpen,
  onClose,
  onSave,
  onDelete,
  editingGroup,
}: GroupModalProps) {
  const [name, setName] = useState('');
  const [sortOrder, setSortOrder] = useState(0);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    if (editingGroup) {
      setName(editingGroup.name);
      setSortOrder(editingGroup.sort_order || 0);
    } else {
      setName('');
      setSortOrder(0);
    }
  }, [editingGroup, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setIsSaving(true);
    try {
      await onSave({ name: name.trim(), sort_order: sortOrder });
      onClose();
    } catch (error) {
      console.error('Failed to save group:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!onDelete || !editingGroup) return;
    if (!confirm(`Delete group "${editingGroup.name}"? Monitors in this group will become ungrouped.`)) return;

    setIsDeleting(true);
    try {
      await onDelete();
      onClose();
    } catch (error) {
      console.error('Failed to delete group:', error);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="sm">
      <form onSubmit={handleSubmit}>
        <ModalHeader onClose={onClose}>
          {editingGroup ? 'edit group' : 'new group'}
        </ModalHeader>

        <div className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Group Name
            </label>
            <Input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
              className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
              placeholder="Production Servers"
            />
          </div>

          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Sort Order
            </label>
            <Input
              type="number"
              value={sortOrder}
              onChange={(e) => setSortOrder(Number(e.target.value))}
              className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
              placeholder="0"
            />
            <p className="text-[10px] text-terminal-muted mt-1">
              Lower numbers appear first
            </p>
          </div>
        </div>

        <ModalFooter>
          <div className="flex gap-3 w-full">
            {editingGroup && onDelete && (
              <Button
                type="button"
                onClick={handleDelete}
                disabled={isDeleting || isSaving}
                variant="outline"
                className="bg-terminal-red/10 border-terminal-red/50 text-terminal-red hover:bg-terminal-red/20"
              >
                {isDeleting ? <Spinner size="sm" /> : 'Delete'}
              </Button>
            )}
            <Button
              type="button"
              onClick={onClose}
              variant="outline"
              className="flex-1 bg-terminal-surface border-terminal-border text-terminal-muted hover:text-terminal-text"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isSaving || isDeleting}
              className="flex-1 bg-terminal-green text-terminal-bg font-bold hover:opacity-90"
            >
              {isSaving ? <Spinner size="sm" /> : 'Save'}
            </Button>
          </div>
        </ModalFooter>
      </form>
    </Modal>
  );
}
