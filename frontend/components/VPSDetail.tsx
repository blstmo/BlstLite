import { useEffect, useRef } from 'react';
import { VPSBackend } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { 
  AlertCircle, 
  Clock, 
  Terminal, 
  Key, 
  ArrowLeft,
  Server,
  Network,
  Globe,
  Copy
} from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { CountdownTimer } from '@/components/countdown-timer';
import { toast } from 'sonner';

interface VPSDetailProps {
  vps: VPSBackend;
  onClose: () => void;
}

export default function VPSDetail({ vps, onClose }: VPSDetailProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const backendUrl = process.env.NEXT_PUBLIC_API_URL;
  const backendHost = backendUrl ? backendUrl.split('//')[1].split(':')[0] : '';

  useEffect(() => {
    if (!backendUrl) {
      console.error('Backend URL not configured');
      return;
    }

    const wsPort = vps.vnc_port + 1000;
    
    const vncParams = new URLSearchParams({
      autoconnect: '1',
      host: backendHost,
      port: wsPort.toString(),
      resize: 'scale',
      quality: '6',
      reconnect: 'true',
    });

    if (iframeRef.current) {
      const novncUrl = `${backendUrl}/novnc/vnc.html?${vncParams.toString()}`;
      console.log('Connecting to VNC:', novncUrl);
      iframeRef.current.src = novncUrl;
    }
  }, [vps, backendUrl, backendHost]);

  const copyToClipboard = (text: string, description: string) => {
    navigator.clipboard.writeText(text);
    toast.success(`${description} copied to clipboard`);
  };

  const sshCommand = `ssh root@${backendHost} -p ${vps.ssh_port}`;

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'success';
      case 'creating':
        return 'warning';
      case 'stopped':
        return 'destructive';
      default:
        return 'secondary';
    }
  };

  return (
    <div className="space-y-6 p-6 bg-background rounded-lg shadow-lg">
      <div className="flex justify-between items-center">
        <div className="space-y-2">
          <h2 className="text-3xl font-bold">{vps.name}</h2>
          <div className="flex items-center gap-4">
            <Badge variant={getStatusColor(vps.status)}>
              {vps.status}
            </Badge>
            <span className="text-muted-foreground text-sm">
              Image: {vps.image_type}
            </span>
          </div>
        </div>
        <Button variant="outline" onClick={onClose} className="flex items-center gap-2">
          <ArrowLeft className="h-4 w-4" />
          Back to List
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Alert>
          <Globe className="h-5 w-5" />
          <AlertTitle>Hostname Details</AlertTitle>
          <AlertDescription className="mt-2 space-y-2">
            <div className="flex items-center justify-between bg-muted p-2 rounded">
              <code className="text-sm">{vps.hostname}</code>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => copyToClipboard(vps.hostname, 'Hostname')}
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </AlertDescription>
        </Alert>

        <Alert>
          <Key className="h-5 w-5" />
          <AlertTitle>System Credentials</AlertTitle>
          <AlertDescription>
            <div className="mt-2 space-y-2 bg-muted p-3 rounded font-mono text-sm">
              <div className="flex justify-between items-center">
                <span>Username: root</span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => copyToClipboard('root', 'Username')}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
              <div className="flex justify-between items-center">
                <span>Password: {vps.password}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => copyToClipboard(vps.password, 'Password')}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </AlertDescription>
        </Alert>

        <Alert variant="warning">
          <Clock className="h-5 w-5" />
          <AlertTitle>Expiration</AlertTitle>
          <AlertDescription>
            <p className="mt-2 font-semibold">
              Expires at {new Date(vps.expires_at).toLocaleString()}
            </p>
            <CountdownTimer expiresAt={vps.expires_at} />
          </AlertDescription>
        </Alert>

        <Alert>
          <Network className="h-5 w-5" />
          <AlertTitle>SSH Connection</AlertTitle>
          <AlertDescription>
            <div className="mt-2 bg-muted p-3 rounded font-mono text-sm">
              <div className="flex justify-between items-center">
                <code>{sshCommand}</code>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => copyToClipboard(sshCommand, 'SSH command')}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </AlertDescription>
        </Alert>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Terminal className="h-5 w-5" />
            VNC Console
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="w-full h-[600px] bg-black rounded-lg overflow-hidden shadow-inner">
            <iframe
              ref={iframeRef}
              className="w-full h-full border-0"
              title={`VNC Console - ${vps.name}`}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            System Details
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
            <div>
              <p className="font-medium text-muted-foreground">VNC Port</p>
              <p>{vps.vnc_port}</p>
            </div>
            <div>
              <p className="font-medium text-muted-foreground">SSH Port</p>
              <p>{vps.ssh_port}</p>
            </div>
            <div>
              <p className="font-medium text-muted-foreground">Created At</p>
              <p>{new Date(vps.created_at).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}