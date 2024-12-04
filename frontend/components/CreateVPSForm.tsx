import React, { useState, useEffect } from 'react';
import { VPS } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card } from '@/components/ui/card';
import { ChevronDown, ChevronUp, Shuffle } from 'lucide-react';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

interface CreateVPSFormProps {
  onSuccess: (vps: VPS) => void;
}

interface OSImage {
  id: string;
  name: string;
  imagePath: string;
  category: string;
  distro: string;
  version: number;
}

export default function CreateVPSForm({ onSuccess }: CreateVPSFormProps) {
  const [name, setName] = useState('');
  const [hostname, setHostname] = useState('');
  const [selectedImage, setSelectedImage] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [availableImages, setAvailableImages] = useState<OSImage[]>([]);
  
  // List of adjectives and nouns for random name generation
  const adjectives = ['swift', 'brave', 'mighty', 'cosmic', 'stellar', 'noble', 'rapid', 'clever', 'nimble', 'radiant'];
  const nouns = ['falcon', 'phoenix', 'dragon', 'titan', 'nexus', 'pulse', 'vertex', 'cipher', 'beacon', 'nova'];
  
  const generateRandomName = () => {
    const adjective = adjectives[Math.floor(Math.random() * adjectives.length)];
    const noun = nouns[Math.floor(Math.random() * nouns.length)];
    const randomNum = Math.floor(Math.random() * 1000);
    const generatedName = `${adjective}-${noun}-${randomNum}`;
    setName(generatedName);
    setHostname(`${generatedName}.vps.local`);
  };

  // Rest of the helper functions remain the same
  const parseVersion = (version: string): number => {
    const match = version.match(/\d+(\.\d+)?/);
    return match ? parseFloat(match[0]) : 0;
  };

  const getOSDetails = (osId: string): OSImage => {
    const [distro, version] = osId.split('-');
    const displayNames: Record<string, { name: string; category: string }> = {
      'ubuntu-24.04': { name: 'Ubuntu 24.04 (Noble)', category: 'Ubuntu' },
      'ubuntu-22.04': { name: 'Ubuntu 22.04 LTS', category: 'Ubuntu' },
      'ubuntu-20.04': { name: 'Ubuntu 20.04 LTS', category: 'Ubuntu' },
      'debian-12': { name: 'Debian 12 (Bookworm)', category: 'Debian' },
      'debian-11': { name: 'Debian 11 (Bullseye)', category: 'Debian' },
      'fedora-40': { name: 'Fedora 40', category: 'Fedora' },
      'fedora-38': { name: 'Fedora 38', category: 'Fedora' },
      'almalinux-9': { name: 'AlmaLinux 9', category: 'Alma Linux' },
      'almalinux-8': { name: 'AlmaLinux 8', category: 'Alma Linux' },
      'rocky-9': { name: 'Rocky Linux 9', category: 'Rocky Linux' },
      'rocky-8': { name: 'Rocky Linux 8', category: 'Rocky Linux' },
      'centos-9': { name: 'CentOS Stream 9', category: 'Centos Linux' },
      'centos-7': { name: 'CentOS 7', category: 'Centos Linux' },
    };

    const details = displayNames[osId] || {
      name: `${distro.charAt(0).toUpperCase() + distro.slice(1)} ${version || ''}`.trim(),
      category: 'Other'
    };

    return {
      id: osId,
      name: details.name,
      category: details.category,
      imagePath: `/images/os/${distro}.png`,
      distro: distro,
      version: parseVersion(version || '0')
    };
  };

  useEffect(() => {
    const fetchImages = async () => {
      try {
        const response = await fetch('/api/vps/images');
        if (!response.ok) throw new Error('Failed to fetch available images');
        
        const imageIds: string[] = await response.json();
        const formattedImages = imageIds.map(getOSDetails);
        
        const categoryOrder = ['Ubuntu', 'Debian', 'Fedora', 'Enterprise Linux', 'Other'];
        const enterpriseDistroOrder = ['almalinux', 'rocky', 'centos'];

        formattedImages.sort((a, b) => {
          const categoryDiff = categoryOrder.indexOf(a.category) - categoryOrder.indexOf(b.category);
          if (categoryDiff !== 0) return categoryDiff;

          if (a.category === 'Enterprise Linux') {
            const distroOrderA = enterpriseDistroOrder.indexOf(a.distro);
            const distroOrderB = enterpriseDistroOrder.indexOf(b.distro);
            
            if (distroOrderA === distroOrderB) {
              return b.version - a.version;
            }
            return distroOrderA - distroOrderB;
          }
          return b.version - a.version;
        });
        
        setAvailableImages(formattedImages);
        if (formattedImages.length > 0) {
          setSelectedImage(formattedImages[0].id);
        }
      } catch (err) {
        console.error('Failed to fetch images:', err);
        setError('Failed to load available images');
      }
    };

    fetchImages();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const response = await fetch('/api/vps/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          name, 
          hostname,
          image_type: selectedImage 
        }),
      });

      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || 'Failed to create VPS');
      }

      onSuccess(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create VPS');
    } finally {
      setLoading(false);
    }
  };

  const groupedImages = availableImages.reduce((groups, os) => {
    if (!groups[os.category]) {
      groups[os.category] = [];
    }
    groups[os.category].push(os);
    return groups;
  }, {} as Record<string, OSImage[]>);

  return (
    <form onSubmit={handleSubmit} className="space-y-6 max-w-3xl mx-auto">
      <div className="space-y-6">
        <div className="space-y-4">
          <div className="flex justify-between items-center">
            <Label htmlFor="name">Instance Name</Label>
            <Button 
              type="button" 
              variant="outline" 
              size="sm"
              onClick={generateRandomName}
              className="flex items-center gap-2"
            >
              <Shuffle className="w-4 h-4" />
              Generate Name
            </Button>
          </div>
          <Input
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-instance"
            required
            disabled={loading}
            minLength={3}
            maxLength={50}
            pattern="[a-zA-Z0-9-]+"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="hostname">Hostname</Label>
          <Input
            id="hostname"
            value={hostname}
            onChange={(e) => setHostname(e.target.value)}
            placeholder="my-instance.vps.local"
            required
            disabled={loading}
          />
        </div>

        <div className="space-y-4">
          <Label>Select Operating System</Label>
          <Accordion type="single" collapsible className="w-full">
            {Object.entries(groupedImages).map(([category, images]) => (
              <AccordionItem key={category} value={category}>
                <AccordionTrigger className="text-lg font-medium">
                  {category}
                </AccordionTrigger>
                <AccordionContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 p-2">
                    {images.map((os) => (
                      <Card
                        key={os.id}
                        className={`p-4 cursor-pointer transition-all hover:ring-2 hover:ring-primary ${
                          selectedImage === os.id ? 'ring-2 ring-primary bg-primary/5' : ''
                        }`}
                        onClick={() => setSelectedImage(os.id)}
                      >
                        <div className="flex items-center space-x-4">
                          <div className="w-10 h-10 relative flex-shrink-0">
                            <img
                              src={os.imagePath}
                              alt={os.name}
                              className="w-full h-full object-contain"
                              onError={(e) => {
                                const img = e.target as HTMLImageElement;
                                img.src = '/images/os/default.png';
                              }}
                            />
                          </div>
                          <span className="text-sm font-medium">{os.name}</span>
                        </div>
                      </Card>
                    ))}
                  </div>
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" disabled={loading} className="w-full">
        {loading ? 'Creating...' : 'Create VPS'}
      </Button>
    </form>
  );
}