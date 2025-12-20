import { useState, useEffect } from 'react';
import { Modal, ModalHeader, ModalFooter } from '@/components/ui/modal';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { cn } from '@/lib/utils';
import { TAG_COLORS } from '@/types';
import type { Tag } from '@/types';

interface TagModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: { name: string; color: string }) => Promise<void>;
  onDelete?: () => Promise<void>;
  editingTag?: Tag | null;
}

export function TagModal({
  isOpen,
  onClose,
  onSave,
  onDelete,
  editingTag,
}: TagModalProps) {
  const [name, setName] = useState('');
  const [color, setColor] = useState(TAG_COLORS[0]);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    if (editingTag) {
      setName(editingTag.name);
      setColor(editingTag.color);
    } else {
      setName('');
      setColor(TAG_COLORS[0]);
    }
  }, [editingTag, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setIsSaving(true);
    try {
      await onSave({ name: name.trim(), color });
      onClose();
    } catch (error) {
      console.error('Failed to save tag:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!onDelete || !editingTag) return;
    if (!confirm(`Delete tag "${editingTag.name}"?`)) return;

    setIsDeleting(true);
    try {
      await onDelete();
      onClose();
    } catch (error) {
      console.error('Failed to delete tag:', error);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="sm">
      <form onSubmit={handleSubmit}>
        <ModalHeader onClose={onClose}>
          {editingTag ? 'edit tag' : 'new tag'}
        </ModalHeader>

        <div className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Tag Name
            </label>
            <Input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoFocus
              className="w-full bg-terminal-surface border-terminal-border text-terminal-text"
              placeholder="production"
            />
          </div>

          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Color
            </label>
            <div className="flex flex-wrap gap-2">
              {TAG_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setColor(c)}
                  className={cn(
                    'w-8 h-8 rounded transition-all',
                    color === c
                      ? 'ring-2 ring-white ring-offset-2 ring-offset-terminal-bg scale-110'
                      : 'hover:scale-105'
                  )}
                  style={{ backgroundColor: c }}
                  title={c}
                />
              ))}
            </div>
          </div>

          {/* Preview */}
          <div>
            <label className="block text-[10px] uppercase text-terminal-muted tracking-widest mb-2">
              Preview
            </label>
            <div className="flex items-center gap-2">
              <span
                className="text-[10px] px-2 py-1 rounded font-semibold"
                style={{
                  background: `${color}30`,
                  color: color,
                  border: `1px solid ${color}`,
                }}
              >
                {name || 'tag name'}
              </span>
            </div>
          </div>
        </div>

        <ModalFooter>
          <div className="flex gap-3 w-full">
            {editingTag && onDelete && (
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
