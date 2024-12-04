# Current Bugs & Issues

Developed by blstmo @ <a href="https://blstmo.com">blstmo.com</a>  

Discord: blstmo  
Join our community: <a href="https://discord.gg/wkgx7Bf2Yt">https://discord.gg/wkgx7Bf2Yt</a>  

# Changelog

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

## Hypervisor Bugs  
- ~~Hypervisor doesn't remove all VPS instances on crash or stop~~  
- ~~No port checking on startup, preventing VPS restart when hypervisor stops~~  
- Network speed limiting not implemented  

## Frontend Bugs  
- ~~Current instances list not functioning~~  

## System Issues  
- No limit on number of hypervisors, can lead to resource overuse  

<img src="/images/manage1.png" alt="Manage 1">  
<img src="/images/manage2.png" alt="Manage 2">  
<img src="/images/manage3.png" alt="Manage 3">