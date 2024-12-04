'use client'

import { useEffect, useRef, useState } from 'react';
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
  Copy,
  AlertTriangle,
  RefreshCcw,
  Loader2,
  Play,
  Square
} from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { CountdownTimer } from '@/components/countdown-timer';
import { restartVPS, startVPS, stopVPS, getVPSDetails } from '@/app/actions';

interface VPSDetailProps {
  vps: VPSBackend;
  onClose: () => void;
}

export default function VPSDetail({ vps: initialVPS, onClose }: VPSDetailProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const backendUrl = process.env.NEXT_PUBLIC_API_URL;
  const backendHost = backendUrl ? backendUrl.split('//')[1].split(':')[0] : '';
  const [copyAlert, setCopyAlert] = useState<string | null>(null);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isStarting, setIsStarting] = useState(false);
  const [isStopping, setIsStopping] = useState(false);
  const [vps, setVPS] = useState<VPSBackend>(initialVPS);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const pollInterval = setInterval(async () => {
      try {
        const updatedVPS = await getVPSDetails(vps.id);
        setVPS(updatedVPS);
        setError(null);
      } catch (err) {
        console.error('Failed to fetch VPS status:', err);
        setError('Failed to update VPS status');
      }
    }, 5000);

    return () => clearInterval(pollInterval);
  }, [vps.id]);

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

    if (iframeRef.current && vps.status === 'running') {
      const novncUrl = `${backendUrl}/novnc/vnc.html?${vncParams.toString()}`;
      console.log('Connecting to VNC:', novncUrl);
      iframeRef.current.src = novncUrl;
    }
  }, [vps.status, vps.vnc_port, backendUrl, backendHost]);

  useEffect(() => {
    if (copyAlert) {
      const timer = setTimeout(() => setCopyAlert(null), 2000);
      return () => clearTimeout(timer);
    }
  }, [copyAlert]);

  const copyToClipboard = (text: string, description: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopyAlert(description);
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  };

  const handleRestart = async () => {
    try {
      setIsRestarting(true);
      setError(null);
      await restartVPS(vps.id);
      setCopyAlert('VPS restart initiated');
    } catch (error) {
      console.error('Failed to restart VPS:', error);
      setError('Failed to restart VPS');
    } finally {
      setIsRestarting(false);
    }
  };

  const handleStart = async () => {
    try {
      setIsStarting(true);
      setError(null);
      await startVPS(vps.id);
      setCopyAlert('VPS start initiated');
    } catch (error) {
      console.error('Failed to start VPS:', error);
      setError('Failed to start VPS');
    } finally {
      setIsStarting(false);
    }
  };

  const handleStop = async () => {
    try {
      setIsStopping(true);
      setError(null);
      await stopVPS(vps.id);
      setCopyAlert('VPS stop initiated');
    } catch (error) {
      console.error('Failed to stop VPS:', error);
      setError('Failed to stop VPS');
    } finally {
      setIsStopping(false);
    }
  };

  const sshCommand = `ssh root@${backendHost} -p ${vps.ssh_port}`;

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'success';
      case 'creating':
        return 'warning';
      case 'stopped':
      case 'stopping':
        return 'destructive';
      case 'starting':
      case 'restarting':
        return 'warning';
      case 'failed':
        return 'destructive';
      default:
        return 'secondary';
    }
  };

  return (
    <div className="space-y-6 p-6 bg-background rounded-lg shadow-lg relative">
      {copyAlert && (
        <div className="absolute top-4 right-4 bg-green-500 text-white px-4 py-2 rounded-md animate-in fade-in slide-in-from-top-1">
          {copyAlert} copied to clipboard
        </div>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex justify-between items-center">
        <div className="space-y-2">
          <h2 className="text-3xl font-bold tracking-tight">{vps.name}</h2>
          <div className="flex items-center gap-4">
            <Badge variant={getStatusColor(vps.status)} className="capitalize">
              {vps.status}
            </Badge>
            <span className="text-muted-foreground text-sm">
              Image: {vps.image_type}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {vps.status === 'stopped' && (
            <Button
              variant="outline"
              onClick={handleStart}
              disabled={isStarting}
              className="flex items-center gap-2"
            >
              {isStarting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Play className="h-4 w-4" />
              )}
              {isStarting ? 'Starting...' : 'Start'}
            </Button>
          )}
          
          {vps.status === 'running' && (
            <>
              <Button
                variant="outline"
                onClick={handleStop}
                disabled={isStopping}
                className="flex items-center gap-2"
              >
                {isStopping ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Square className="h-4 w-4" />
                )}
                {isStopping ? 'Stopping...' : 'Stop'}
              </Button>
              
              <Button
                variant="outline"
                onClick={handleRestart}
                disabled={isRestarting}
                className="flex items-center gap-2"
              >
                {isRestarting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCcw className="h-4 w-4" />
                )}
                {isRestarting ? 'Restarting...' : 'Restart'}
              </Button>
            </>
          )}
          
          <Button variant="outline" onClick={onClose} className="flex items-center gap-2">
            <ArrowLeft className="h-4 w-4" />
            Back to List
          </Button>
        </div>
      </div>

      {vps.status === 'failed' && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Instance Failed</AlertTitle>
          <AlertDescription>
            This instance has failed to start properly. You may want to delete it and create a new one.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        <Alert>
          <Globe className="h-5 w-5" />
          <AlertTitle>Hostname Details</AlertTitle>
          <AlertDescription className="mt-2 space-y-2">
            <div className="flex items-center justify-between bg-muted/50 p-2 rounded">
              <code className="text-sm font-mono">{vps.hostname}</code>
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
            <div className="mt-2 space-y-2 bg-muted/50 p-3 rounded font-mono text-sm">
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
          <AlertTitle>Instance Expiration</AlertTitle>
          <AlertDescription>
            <p className="mt-2 font-medium">
              Expires at {new Date(vps.expires_at).toLocaleString()}
            </p>
            <CountdownTimer expiresAt={vps.expires_at} />
          </AlertDescription>
        </Alert>

        <Alert>
          <Network className="h-5 w-5" />
          <AlertTitle>SSH Connection</AlertTitle>
          <AlertDescription>
            <div className="mt-2 bg-muted/50 p-3 rounded font-mono text-sm">
              <div className="flex justify-between items-center break-all">
                <code>{sshCommand}</code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="ml-2 flex-shrink-0"
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
          <div className="w-full aspect-video bg-black rounded-lg overflow-hidden shadow-inner">
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
              <p className="font-mono">{vps.vnc_port}</p>
            </div>
            <div>
              <p className="font-medium text-muted-foreground">SSH Port</p>
              <p className="font-mono">{vps.ssh_port}</p>
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