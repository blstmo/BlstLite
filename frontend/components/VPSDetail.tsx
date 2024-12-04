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
  Square,
  Trash2
} from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { CountdownTimer } from '@/components/countdown-timer';
import { 
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { restartVPS, startVPS, stopVPS, getVPSDetails, deleteVPS } from '@/app/actions';
import ResourceMonitoring from './ResourceChart';

interface VPSDetailProps {
  vps: VPSBackend;
  onClose: () => void;
  onDelete?: () => void;
}

export default function VPSDetail({ vps: initialVPS, onClose, onDelete }: VPSDetailProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const backendUrl = process.env.NEXT_PUBLIC_API_URL;
  const backendHost = backendUrl ? backendUrl.split('//')[1].split(':')[0] : '';
  const [copyAlert, setCopyAlert] = useState<string | null>(null);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isStarting, setIsStarting] = useState(false);
  const [isStopping, setIsStopping] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
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

  const handleDelete = async () => {
    try {
      setIsDeleting(true);
      setError(null);
      await deleteVPS(vps.id);
      onDelete?.();
    } catch (error) {
      console.error('Failed to delete VPS:', error);
      setError('Failed to delete VPS');
    } finally {
      setIsDeleting(false);
      setShowDeleteDialog(false);
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
        <div className="absolute top-4 right-4 bg-green-500/90 backdrop-blur-sm text-white px-4 py-2 rounded-md animate-in fade-in slide-in-from-top-1 shadow-lg">
          {copyAlert} copied to clipboard
        </div>
      )}

      {error && (
        <Alert variant="destructive" className="bg-red-500/10 border-red-500/50">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex justify-between items-center bg-primary/5 p-4 rounded-lg">
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
  {(vps.status === 'stopped' || vps.status === 'failed') && (
    <Button
      variant="outline"
      onClick={handleStart}
      disabled={isStarting}
      className="bg-green-500/10 hover:bg-green-500/20 border-green-500/30"
    >
      {isStarting ? (
        <Loader2 className="h-4 w-4 animate-spin mr-2" />
      ) : (
        <Play className="h-4 w-4 mr-2" />
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
        className="bg-orange-500/10 hover:bg-orange-500/20 border-orange-500/30"
      >
        {isStopping ? (
          <Loader2 className="h-4 w-4 animate-spin mr-2" />
        ) : (
          <Square className="h-4 w-4 mr-2" />
        )}
        {isStopping ? 'Stopping...' : 'Stop'}
      </Button>
      
      <Button
        variant="outline"
        onClick={handleRestart}
        disabled={isRestarting}
        className="bg-blue-500/10 hover:bg-blue-500/20 border-blue-500/30"
      >
        {isRestarting ? (
          <Loader2 className="h-4 w-4 animate-spin mr-2" />
        ) : (
          <RefreshCcw className="h-4 w-4 mr-2" />
        )}
        {isRestarting ? 'Restarting...' : 'Restart'}
      </Button>
    </>
  )}
  
  <Button
    variant="destructive"
    onClick={() => setShowDeleteDialog(true)}
    disabled={isDeleting}
    className="bg-red-500/10 hover:bg-red-500/20"
  >
    {isDeleting ? (
      <Loader2 className="h-4 w-4 animate-spin mr-2" />
    ) : (
      <Trash2 className="h-4 w-4 mr-2" />
    )}
    {isDeleting ? 'Deleting...' : 'Delete'}
  </Button>
  
  <Button variant="outline" onClick={onClose}>
    <ArrowLeft className="h-4 w-4 mr-2" />
    Back
  </Button>
</div>
      </div>

      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure you want to delete this VPS?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the VPS
              "{vps.name}" and all associated data.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-red-500 hover:bg-red-600"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {vps.status === 'failed' && (
        <Alert variant="destructive" className="bg-red-500/10 border-red-500/50">
          <AlertTriangle className="h-4 w-4" />
          <AlertTitle>Instance Failed</AlertTitle>
          <AlertDescription>
            This instance has failed to start properly. You may want to delete it and create a new one.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        <Alert className="bg-blue-500/5 border-blue-500/30">
          <Globe className="h-5 w-5 text-blue-500" />
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

        <Alert className="bg-purple-500/5 border-purple-500/30">
          <Key className="h-5 w-5 text-purple-500" />
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

        <Alert variant="warning" className="bg-yellow-500/5 border-yellow-500/30">
          <Clock className="h-5 w-5 text-yellow-500" />
          <AlertTitle>Instance Expiration</AlertTitle>
          <AlertDescription>
            <p className="mt-2 font-medium">
              Expires at {new Date(vps.expires_at).toLocaleString()}
            </p>
            <CountdownTimer expiresAt={vps.expires_at} />
          </AlertDescription>
        </Alert>

        <Alert className="bg-green-500/5 border-green-500/30">
          <Network className="h-5 w-5 text-green-500" />
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

      <Card className="bg-gradient-to-b from-background to-background/90 border-primary/20">
      <CardHeader className="border-b border-primary/10">
          <CardTitle className="flex items-center gap-2 text-primary">
            <Terminal className="h-5 w-5" />
            VNC Console
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="w-full aspect-video bg-black rounded-lg overflow-hidden shadow-inner">
            {vps.status === 'running' ? (
              <iframe
                ref={iframeRef}
                className="w-full h-full border-0"
                title={`VNC Console - ${vps.name}`}
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-muted-foreground">
                <p>VNC console is only available when the instance is running</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
      
      <Card className="bg-gradient-to-b from-background to-background/90 border-primary/20">
        <CardHeader className="border-b border-primary/10">
          <CardTitle className="flex items-center gap-2 text-primary">
            <Server className="h-5 w-5" />
            System Details
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-6 text-sm">
            <div className="space-y-1">
              <p className="font-medium text-muted-foreground">VNC Port</p>
              <p className="font-mono bg-muted/50 p-2 rounded">{vps.vnc_port}</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-muted-foreground">SSH Port</p>
              <p className="font-mono bg-muted/50 p-2 rounded">{vps.ssh_port}</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-muted-foreground">Created At</p>
              <p className="bg-muted/50 p-2 rounded">{new Date(vps.created_at).toLocaleString()}</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-muted-foreground">Status</p>
              <p className="bg-muted/50 p-2 rounded capitalize">{vps.status}</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-muted-foreground">Image</p>
              <p className="bg-muted/50 p-2 rounded">{vps.image_type}</p>
            </div>
            <div className="space-y-1">
              <p className="font-medium text-muted-foreground">ID</p>
              <p className="font-mono bg-muted/50 p-2 rounded text-xs">{vps.id}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="bg-gradient-to-b from-background to-background/90 border-primary/20">
        <CardHeader className="border-b border-primary/10">
          <CardTitle className="text-primary flex items-center gap-2">
            <AlertCircle className="h-5 w-5" />
            Resource Monitoring
          </CardTitle>
        </CardHeader>
        <CardContent>
          {vps.status === 'running' ? (
            <ResourceMonitoring 
              vpsId={vps.id} 
              isRunning={vps.status === 'running'} 
            />
          ) : (
            <div className="flex items-center justify-center h-48 text-muted-foreground">
              <p>Resource monitoring is only available when the instance is running</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
