using OpenIddict.Server.AspNetCore;

WebApplicationBuilder builder = WebApplication.CreateBuilder(args);

if (!builder.Environment.IsProduction())
{
    builder.Configuration.AddJsonFile("appsettings.local.json", optional: true, reloadOnChange: true);
}

builder.CreateUmbracoBuilder()
    .AddBackOffice()
    .AddWebsite()
    .AddComposers()
    .Build();

// This test instance runs HTTP-only (no self-signed cert to trust), so the OAuth2
// token endpoint must be reachable without HTTPS transport.
if (builder.Environment.IsDevelopment())
{
    builder.Services.Configure<OpenIddictServerAspNetCoreOptions>(options =>
    {
        options.DisableTransportSecurityRequirement = true;
    });
}

WebApplication app = builder.Build();

await app.BootUmbracoAsync();

app.UseUmbraco()
    .WithMiddleware(u =>
    {
        u.UseBackOffice();
        u.UseWebsite();
    })
    .WithEndpoints(u =>
    {
        u.UseBackOfficeEndpoints();
        u.UseWebsiteEndpoints();
    });

await app.RunAsync();
