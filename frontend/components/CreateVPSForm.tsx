// components/CreateVPSForm.tsx
import { useState } from 'react';
import { VPS } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';

interface CreateVPSFormProps {
  onSuccess: (vps: VPS) => void;
}

export default function CreateVPSForm({ onSuccess }: CreateVPSFormProps) {
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      console.log('Submitting VPS creation with name:', name);

      const response = await fetch('/api/vps/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });

      const data = await response.json();
      console.log('Create VPS response:', data);

      if (!response.ok) {
        throw new Error(data.error || 'Failed to create VPS');
      }

      onSuccess(data);
    } catch (err) {
      console.error('VPS creation error:', err);
      setError(err instanceof Error ? err.message : 'Failed to create VPS');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <Label htmlFor="name">Instance Name</Label>
        <Input
          id="name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="my-ubuntu-vps"
          required
          disabled={loading}
          minLength={3}
          maxLength={50}
          pattern="[a-zA-Z0-9-]+"
        />
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" disabled={loading}>
        {loading ? 'Creating...' : 'Create VPS'}
      </Button>
    </form>
  );
}