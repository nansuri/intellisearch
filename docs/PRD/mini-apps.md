# Mini-Apps Support and Mini-Apps Studio

## Summary Overview
As a user, i want to be able to create my own mini apps, i just need to provide HTML + JS + CSS and the app will be rendered by this app
As a user, i will also be able to see "Mini Apps Studio" which is new page that contain IDE and also "Preview" to manage, edit, delete and modify my mini apps

## Business Details
1. Mini APP entry point is from the "Apps" button on top left, user also will be able to see "Mini Apps Studio" button on their avatar button
2. User will be able to decide to publish their mini app as Public / Private
3. We need to revamp the App button to support a lot of app list
4. When user click on "Mini Apps Studio", they will be redirected to the "Mini Apps Studio" page which begin with List of their Apps
5. User will be able to edit, enhance as well
6. There also a feature where we can ask AI to create the APP by single prompt and requirement
7. There also some APIs that user can use liek for CRUD, AI api (will also utilize AI Handler and Limit), etc.
8. User will be able to see the API list in Mini Apps Studio main page
9. User can also download the AI API as markdown, in case they want to use their own AI

## Technical Details
- Since this feature will be big, i want you to save these HTML, JS, CSS as a file or store in totally different table (decide which one is much more efficient)
- For this feature, always create new .vue for every page especially for the IDE
- for API documentation, store it in database, do not store any details in FE
- FE is only for view, all data stored by Backend
- Front end will render these