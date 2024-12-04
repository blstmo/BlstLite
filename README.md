# Info

Developed by blstmo @ <a href="https://blstmo.com">blstmo.com</a>  

Discord: blstmo  
Join our community: <a href="https://discord.gg/wkgx7Bf2Yt">https://discord.gg/wkgx7Bf2Yt</a>  

<img src="/images/manage1.png" alt="Manage 1">  
<img src="/images/manage2.png" alt="Manage 2">  
<img src="/images/manage3.png" alt="Manage 3">  

# Issues & Bugs

## Hypervisor Bugs  
- ~~Hypervisor doesn't remove all VPS instances on crash or stop~~  
- ~~No port checking on startup, preventing VPS restart when hypervisor stops~~  
- Network speed limiting not implemented  

## Frontend Bugs  
- ~~Current instances list not functioning~~  

## System Issues  
- No limit on number of hypervisors, can lead to resource overuse

# Changelog

# Changelog

## Version 1.1.3 (2024-12-04)
### Added Features
- Added comprehensive network metrics collection for VPS instances
- Implemented real-time network traffic monitoring (RX/TX bytes and packets)
- Added network speed calculations (bytes/sec) for both upload and download
- Enhanced metrics logging for better debugging and monitoring

### Technical Improvements
- Added multiple fallback methods for network statistics collection
- Implemented QMP (QEMU Machine Protocol) based network monitoring
- Added direct process statistics monitoring through /proc filesystem
- Enhanced metrics caching for accurate speed calculations

### Bug Fixes
- Resolved network metrics collection in user-mode networking
- Improved network device detection reliability
- Added better error handling for network statistics collection


## Version 1.1.2 (2024-12-04)
### Added Features
- Implemented VPS power management controls (Start, Stop, Restart)
- Added real-time status polling for VPS instances
- Improved VPS state visualization with status badges
- Enhanced error handling for power management operations

### Bug Fixes
- Fixed VPS status tracking during power state transitions
- Resolved websocket connection issues during power operations
- Improved VNC console reconnection behavior
- Added proper error messaging for failed operations

### UI Improvements
- Added loading states for all power management actions
- Updated status badge colors to reflect different VPS states
- Enhanced error notifications and user feedback
- Improved button visibility based on VPS state

## Version 1.1.0 (2024-12-04)
### Architecture Changes
- Removed frontend API routes in favor of Next.js Server Actions for improved security and performance
- Migrated all API calls to server-side functions
- Added progress tracking for VPS creation process
- Improved error handling and state management

### Changed Components
- CreateVPSForm: Removed direct API calls, added progress tracking
- VPSList: Migrated to server actions, improved error handling
- VPSDetail: Removed frontend API dependencies, added better state management
- All delete/create operations now handled through server actions - delete actions are for future functions

### Security Improvements
- API keys and sensitive configuration now handled server-side only
- Removed client-side exposure of backend API endpoints
- Better error handling and validation on server side

