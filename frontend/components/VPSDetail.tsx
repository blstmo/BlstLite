import { useEffect, useRef } from 'react';
import { VPS } from '@/types/vps';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { AlertCircle, Clock, Terminal, Key, ArrowLeft } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { CountdownTimer } from '@/components/countdown-timer';

interface VPSDetailProps {
  vps: VPS;
  onClose: () => void;
}

export default function VPSDetail({ vps, onClose }: VPSDetailProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    const backendUrl = process.env.NEXT_PUBLIC_API_URL;
    if (!backendUrl) {
      console.error('Backend URL not configured');
      return;
    }

    const backendHost = backendUrl.split('//')[1].split(':')[0];
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
  }, [vps]);

  return (
    <div className="space-y-6 p-6 bg-gray-900 text-gray-100 rounded-lg shadow-lg">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold text-gray-100">{vps.name}</h2>
          <Badge variant={vps.status === 'running' ? 'success' : 'secondary'} className="mt-2">
            {vps.status}
          </Badge>
        </div>
        <Button variant="outline" onClick={onClose} className="flex items-center gap-2 text-gray-300 hover:text-gray-100">
          <ArrowLeft className="h-4 w-4" />
          Back to List
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Alert variant="default" className="bg-gray-800 border-gray-700">
          <Key className="h-5 w-5 text-blue-400" />
          <AlertTitle className="text-blue-400">System Credentials</AlertTitle>
          <AlertDescription>
            <div className="mt-2 bg-gray-700 p-3 rounded-md font-mono text-sm shadow-sm">
              <p><span className="text-gray-400">Username:</span> root</p>
              <p><span className="text-gray-400">Password:</span> {vps.password}</p>
            </div>
          </AlertDescription>
        </Alert>

        <Alert variant="warning" className="bg-yellow-900 border-yellow-800">
          <Clock className="h-5 w-5 text-yellow-400" />
          <AlertTitle className="text-yellow-400">Expiration</AlertTitle>
          <AlertDescription>
            <p className="mt-2 font-semibold text-yellow-200">
              Expires at {new Date(vps.expires_at).toLocaleString()}
            </p>
            <CountdownTimer expiresAt={vps.expires_at} />
          </AlertDescription>
        </Alert>
      </div>

      <Card className="bg-gray-800 border-gray-700">
        <CardHeader className="bg-gray-700">
          <CardTitle className="flex items-center text-xl text-gray-100">
            <Terminal className="h-6 w-6 mr-2 text-gray-400" />
            VNC Console
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="w-full h-[600px] bg-black rounded-b-lg overflow-hidden shadow-inner">
            <iframe
              ref={iframeRef}
              className="w-full h-full border-0"
              title={`VNC Console - ${vps.name}`}
            />
          </div>
        </CardContent>
      </Card>

      <Card className="bg-gray-800 border-gray-700">
        <CardHeader>
          <CardTitle className="text-lg text-gray-100">Connection Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="font-semibold text-gray-400">VNC Port</p>
              <p className="text-gray-200">{vps.vnc_port}</p>
            </div>
            <div>
              <p className="font-semibold text-gray-400">SSH Port</p>
              <p className="text-gray-200">{10000 + vps.vnc_port}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

